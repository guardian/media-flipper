package analysis

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"log"
	"net/http"
	"net/url"
	"reflect"
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

	requestUrl, urlErr := url.ParseRequestURI(r.RequestURI)
	if urlErr != nil {
		log.Print("requestURI could not parse, this should not happen: ", urlErr)
		return
	}
	uuidText := requestUrl.Query().Get("forJob")
	jobContainerId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		log.Printf("Could not parse forJob parameter %s into a UUID: %s", uuidText, uuidErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Invalid forJob parameter"}, w, 400)
		return
	}

	jobStepText := requestUrl.Query().Get("stepId")
	jobStepId, uuidErr := uuid.Parse(jobStepText)

	if uuidErr != nil {
		log.Printf("Could not parse stepId parameter %s into a UUID: %s", jobStepText, uuidErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Invalid stepId parameter"}, w, 400)
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

	jobContainerInfo, containerGetErr := models.JobContainerForId(jobContainerId, h.redisClient)
	if containerGetErr != nil {
		log.Printf("Could not retrieve job container for %s: %s", jobContainerId.String(), containerGetErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "invalid job id"}, w, 400)
		return
	}

	jobStepCopyPtr := jobContainerInfo.FindStepById(jobStepId)
	if jobStepCopyPtr == nil {
		log.Printf("Job container %s does not have any step with the id %s", jobContainerId.String(), jobStepId.String())
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no jobstep with that id in the given job"}, w, 404)
		return
	}

	jobStepCopy := *jobStepCopyPtr

	analysisStep, isAnalysis := jobStepCopy.(*models.JobStepAnalysis)

	if !isAnalysis {
		log.Printf("Expected step %s of job %s to be analysis type but got %s", jobStepId, jobContainerId, reflect.TypeOf(jobStepCopy))
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "identified jobstep was not analysis"}, w, 404)
		return
	}

	analysisStep.StatusValue = models.JOB_COMPLETED
	analysisStep.ResultId = uuid.New()

	newRecord := models.FileFormatInfo{
		Id:             analysisStep.ResultId,
		FormatAnalysis: incoming.Format,
	}

	putErr := models.PutFileFormat(&newRecord, h.redisClient)
	if putErr != nil {
		log.Printf("Could not save record to datastore")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "Could not save record",
		}, w, 500)
		return
	}

	updateErr := jobContainerInfo.UpdateStepById(jobStepId, analysisStep)
	if updateErr != nil {
		log.Printf("Could not set jobstep info for %s in job %s: %s", jobStepId, jobContainerId, updateErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: updateErr.Error(),
		}, w, 500)
		return
	}
	jobSaveErr := jobContainerInfo.Store(h.redisClient)
	if jobSaveErr != nil {
		log.Printf("Could not save record to datastore")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "Could not save record",
		}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]string{"status": "ok", "detail": "Record saved", "entryId": newRecord.Id.String()}, w, 200)

}
