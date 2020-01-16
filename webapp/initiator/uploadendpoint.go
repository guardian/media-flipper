package initiator

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/jobs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type UploadEndpointHandler struct {
	config      *helpers.Config
	redisClient *redis.Client
}

func (h UploadEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != "POST" {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "wrong method type"}, w, 405)
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	uuidText := requestUrl.Query().Get("forJob")
	jobId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		log.Printf("Could not parse forJob parameter %s into a UUID: %s", uuidText, uuidErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Invalid jobId parameter"}, w, 400)
		return
	}

	log.Printf("File upload requested")

	if r.Header.Get("Content-Type") != "application/octet-stream" {
		log.Printf("Incorrect content type, expected application/octet-stream got %s", r.Header.Get("Content-Type"))
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_request", "Need application/octet-stream content"}, w, 415)
		return
	}

	jobRecord, jobErr := jobs.GetJobForId(jobId, h.redisClient)
	if jobErr != nil {
		log.Printf("Could not retrieve job record for %s: %s", jobId, jobErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not retrieve record"}, w, 500)
		return
	}

	if jobRecord == nil {
		log.Printf("No job present for %s", jobId)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "Job does not exist"}, w, 404)
		return
	}

	log.Print("Got job record: ", *jobRecord)
	//
	//if jobRecord.MediaFile != "" {
	//	log.Printf("Job already has a file! - %s", jobRecord.MediaFile)
	//	helpers.WriteJsonContent(helpers.GenericErrorResponse{"error","Job already has a file"}, w, 400)
	//	return
	//}

	uploadFileBasepath := h.config.Scratch.LocalPath

	fp, fpErr := ioutil.TempFile(uploadFileBasepath, "upload*")
	if fpErr != nil {
		log.Printf("Could not create tempfile at %s: %s", uploadFileBasepath, fpErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not write data server-side"}, w, 500)
		return
	}

	log.Printf("Uploading to %s", fp.Name())
	bytesCopied, writeErr := io.Copy(fp, r.Body)

	if writeErr != nil {
		log.Printf("Could not write data to %s: %s", fp.Name())
	}

	jobRecord.MediaFile = fp.Name()
	jobUpdateErr := jobs.PutJob(jobRecord, h.redisClient)
	if jobUpdateErr != nil {
		log.Printf("ERROR: Could not update job record: %s", writeErr)
		defer os.Remove(fp.Name())
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not update job"}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{"status": "ok", "receivedBytes": bytesCopied}, w, 200)

}
