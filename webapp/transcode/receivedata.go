package transcode

import (
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/results"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	"github.com/guardian/mediaflipper/webapp/models"
	"io/ioutil"
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
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return //response should already be sent here
	}

	requestUrl, urlErr := url.ParseRequestURI(r.RequestURI)
	if urlErr != nil {
		log.Print("requestURI could not parse, this should not happen: ", urlErr)
		return
	}
	uuidText := requestUrl.Query().Get("forJob")
	jobContainerId, uuidErr := uuid.Parse(uuidText)

	jobStepText := requestUrl.Query().Get("stepId")
	jobStepId, uuidErr := uuid.Parse(jobStepText)

	if uuidErr != nil {
		log.Printf("Could not parse stepId parameter %s into a UUID: %s", jobStepText, uuidErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Invalid stepId parameter"}, w, 400)
		return
	}

	var incoming results.TranscodeResult
	rawContent, _ := ioutil.ReadAll(r.Body)
	parseErr := json.Unmarshal(rawContent, &incoming)

	if parseErr != nil {
		log.Printf("Could not understand content body: %s", parseErr)
		log.Printf("Offending data was: %s", string(rawContent))
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_request", parseErr.Error()}, w, 400)
		return
	}

	var fileEntry *models.FileEntry = nil
	if incoming.OutFile != "" {
		f, fileEntryErr := models.NewFileEntry(incoming.OutFile, jobContainerId, models.TYPE_TRANSCODE)
		if fileEntryErr != nil {
			log.Printf("Could not get information for incoming thumbnail %s: %s", spew.Sprint(incoming), fileEntryErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not get file info"}, w, 500)
			return
		}
		fileEntry = &f
		storErr := fileEntry.Store(h.redisClient)
		if storErr != nil {
			log.Printf("Could not store file entry: %s", storErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", storErr.Error()}, w, 500)
			return
		}
	}

	completionChan := make(chan bool)

	whenQueueReady := func(waitErr error) {
		if waitErr != nil {
			log.Printf("queue wait failed: %s", waitErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "queue wait failed, see the logs"}, w, 500)
			completionChan <- false
			return
		}

		jobContainerInfo, containerGetErr := models.JobContainerForId(jobContainerId, h.redisClient)
		if containerGetErr != nil {
			log.Printf("Could not retrieve job container for %s: %s", jobContainerId.String(), containerGetErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Invalid job id"}, w, 400)
			completionChan <- false
			return
		}

		jobStepCopyPtr := jobContainerInfo.FindStepById(jobStepId)
		if jobStepCopyPtr == nil {
			log.Printf("Job container %s does not have any step with the id %s", jobContainerId.String(), jobStepId.String())
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no jobstep with that id in the given job"}, w, 404)
			completionChan <- false
			return
		}

		tcStep, isTc := (*jobStepCopyPtr).(*models.JobStepTranscode)
		if !isTc {
			log.Printf("Job step was not transcode type, got %s", reflect.TypeOf(jobStepCopyPtr))
			helpers.WriteJsonContent(helpers.GenericErrorResponse{
				Status: "bad_request",
				Detail: "job step was not transcode",
			}, w, 400)
			completionChan <- false
			return
		}

		if fileEntry != nil {
			tcStep.ResultId = &(fileEntry.Id)
		}

		var updatedStep models.JobStep
		if incoming.ErrorMessage != "" {
			updatedStep = tcStep.WithNewStatus(models.JOB_FAILED, &incoming.ErrorMessage)
		} else {
			updatedStep = tcStep.WithNewStatus(models.JOB_COMPLETED, nil)
		}

		updateErr := jobContainerInfo.UpdateStepById(updatedStep.StepId(), updatedStep)
		if updateErr != nil {
			log.Printf("Could not update job container with new step: %s", updateErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not update job container"}, w, 500)
			completionChan <- false
			return
		}

		log.Printf("Storing updated container...")
		spew.Dump(jobContainerInfo)

		storErr := jobContainerInfo.Store(h.redisClient)
		if storErr != nil {
			log.Printf("Could not store job container: %s", storErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not store updated content"}, w, 200)
			completionChan <- false
			return
		}

		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "result saved"}, w, 200)
		completionChan <- true
	}

	jobrunner.WhenQueueAvailable(h.redisClient, jobrunner.RUNNING_QUEUE, whenQueueReady, true)
	<-completionChan
}
