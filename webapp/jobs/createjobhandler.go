package jobs

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	models2 "github.com/guardian/mediaflipper/common/models"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
)

type CreateJobHandler struct {
	RedisClient *redis.Client
	K8Client    *kubernetes.Clientset
	TemplateMgr *models2.JobTemplateManager
}

func (h CreateJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	textBody, textReadErr := ioutil.ReadAll(r.Body)
	if textReadErr != nil {
		log.Print("Could not read request body content ", textReadErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not read body content"}, w, 500)
		return
	}

	var rq JobRequest
	unmarshalErr := json.Unmarshal(textBody, &rq)
	if unmarshalErr != nil {
		log.Print("Could not unmarshal request body ", unmarshalErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Invalid json request body"}, w, 400)
		return
	}

	newEntry, createErr := h.TemplateMgr.NewJobContainer(rq.SettingsId, rq.JobTemplateId)
	if createErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"server_error", createErr.Error()}, w, 500)
		return
	}

	jobErr := newEntry.Store(h.RedisClient)
	if jobErr != nil {
		log.Print("Could not save new job: ", jobErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not save record"}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]string{"status": "ok", "jobContainerId": newEntry.Id.String()}, w, 201)
}
