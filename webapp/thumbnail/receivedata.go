package thumbnail

import (
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
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

	var incoming models.ThumbnailResult
	contentBytes, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("could not read in request body: %s", readErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not read response body"}, w, 400)
		return
	}

	marshalErr := json.Unmarshal(contentBytes, &incoming)
	if marshalErr != nil {
		log.Printf("malformed contant body, could not unmarshal json: %s", marshalErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_input", "Could not parse/marshal json content"}, w, 400)
		return
	}

	var fileEntry models.FileEntry
	if incoming.OutPath != nil {
		f, fileEntryErr := models.NewFileEntry(*incoming.OutPath, jobContainerId, models.TYPE_THUMBNAIL)
		if fileEntryErr != nil {
			log.Printf("Could not get information for incoming thumbnail %s: %s", spew.Sprint(incoming), fileEntryErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not get file info"}, w, 500)
			return
		}
		fileEntry = f
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

		jobStepCopy := *jobStepCopyPtr

		thumbStep, isThumb := jobStepCopy.(*models.JobStepThumbnail)

		if !isThumb {
			log.Printf("Expected step %s of job %s to be thumbnail type but got %s", jobStepId, jobContainerId, reflect.TypeOf(jobStepCopy))
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "identified jobstep was not thumbnail"}, w, 404)
			completionChan <- false
			return
		}

		if fileEntry.ServerPath != "" {
			//we got a valid file
			storErr := fileEntry.Store(h.redisClient)
			if storErr != nil {
				log.Printf("Could not store new file entry: %s", storErr)
				helpers.WriteJsonContent(helpers.GenericErrorResponse{
					Status: "db_error",
					Detail: "could not write file entry to database",
				}, w, 500)
				return
			}
			thumbStep.ResultId = &fileEntry.Id
		}

		var updatedStep models.JobStep
		if incoming.ErrorMessage == nil || *incoming.ErrorMessage == "" {
			updatedStep = thumbStep.WithNewStatus(models.JOB_COMPLETED, nil)
		} else {
			updatedStep = thumbStep.WithNewStatus(models.JOB_FAILED, incoming.ErrorMessage)
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
