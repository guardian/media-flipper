package jobrunner

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"net/url"
)

type ManualCleanupHandler struct {
	redisClient *redis.Client
	k8clientset *kubernetes.Clientset
}

//manually initiate a cleanup operation
func (h ManualCleanupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	ns, nsErr := GetMyNamespace()
	if nsErr != nil {
		log.Printf("ERROR ManualCleanupHandler could not determine namespace: %s", nsErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not determine namespace"}, w, 500)
		return
	}

	parsedUrl, urlErr := url.Parse(r.RequestURI)
	if urlErr != nil {
		log.Printf("ERROR ManualCleanupHandler could not parse the url: %s", urlErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not understand url"}, w, 400)
		return
	}

	stepIdString := parsedUrl.Query().Get("stepId")

	if stepIdString == "" {
		log.Printf("ERROR ManualCleanupHandler could not find a stepId parameter")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "no stepId parameter"}, w, 400)
		return
	}

	stepId, stepIdParseErr := uuid.Parse(stepIdString)
	if stepIdParseErr != nil {
		log.Printf("ERROR ManualCleanupHandler stepId was not valid: %s", stepIdParseErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "stepId was not valid"}, w, 400)
		return
	}
	jobIdString := parsedUrl.Query().Get("jobId")
	if jobIdString == "" {
		log.Printf("ERROR ManualCleanupHandler could not find a jobId parameter")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "no jobId parameter"}, w, 400)
		return
	}

	jobId, jobIdParseErr := uuid.Parse(jobIdString)
	if jobIdParseErr != nil {
		log.Printf("ERROR ManualCleanupHandler jobId was not valid: %s", jobIdParseErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", fmt.Sprintf("jobId was not valid: %s", jobIdParseErr)}, w, 400)
		return
	}

	log.Printf("DEBUG ManualCleanupHandler namespace is %s", ns)
	jobclient := h.k8clientset.BatchV1().Jobs(ns)
	podclient := h.k8clientset.CoreV1().Pods(ns)

	jobInfo, getErr := models.JobContainerForId(jobId, h.redisClient)
	if getErr != nil {
		log.Printf("ERROR ManualCleanupHandler could not look up job container with id '%s': %s", jobId, getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_err", "could not look up job"}, w, 500)
		return
	}

	if jobInfo == nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no job with that id"}, w, 404)
		return
	}
	step := jobInfo.FindStepById(stepId)
	if step == nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no jobstep with that id in the given job"}, w, 404)
		return
	}

	err := CleanUpJobStep(step, jobclient, podclient, h.redisClient)
	if err != nil {
		log.Printf("ERROR ManualCleanupHandler could not perform cleanup: %s", err)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", err.Error()}, w, 500)
		return
	}
	helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "cleanup successful"}, w, 200)
}
