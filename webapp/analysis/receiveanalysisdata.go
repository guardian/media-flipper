package analysis

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"log"
	"net/http"
	"net/url"
)

type ReceiveData struct {
	redisClient *redis.Client
}

func (h ReceiveData) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if r.Method != "POST" {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "expected POST",
		}, w, 405)
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

	var incoming models.AnalysisResult
	readErr := helpers.ReadJsonBody(r.Body, &incoming)
	if readErr != nil {
		log.Printf("ERROR: Could not parse incoming data to ReceiveAnalysisData: %s", readErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "could not read json",
		}, w, 400)
		return
	}

	newRecord := models.FileFormatInfo{
		ForJob:         jobId,
		FormatAnalysis: incoming.Format,
	}

	putErr := models.PutFileFormat(&newRecord, h.redisClient)
	if putErr != nil {
		log.Printf("Could not save record to datastore")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "Could not save record",
		}, w, 500)
	} else {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "ok",
			Detail: "Record saved",
		}, w, 200)
	}
}
