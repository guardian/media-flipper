package jobs

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"net/url"
)

type JobsEndpoints struct {
	GetHandler    GetJobHandler
	CreateHandler CreateJobHandler
	ListHandler   ListJobHandler
	StatusHandler StatusJobHandler
}

func NewJobsEndpoints(redisClient *redis.Client, k8client *kubernetes.Clientset) JobsEndpoints {
	return JobsEndpoints{
		GetHandler:    GetJobHandler{redisClient},
		CreateHandler: CreateJobHandler{redisClient},
		ListHandler:   ListJobHandler{redisClient},
		StatusHandler: StatusJobHandler{k8client},
	}
}

func (e JobsEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/get", e.GetHandler)
	http.Handle(baseUrlPath+"/new", e.CreateHandler)
	http.Handle(baseUrlPath+"/status", e.StatusHandler)
	http.Handle(baseUrlPath+"", e.ListHandler)
}

type GetJobHandler struct {
	RedisClient *redis.Client
}

func (h GetJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return //error is already output
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)

	jobIdString := requestUrl.Query().Get("jobId")
	jobId, uuidErr := uuid.Parse(jobIdString)
	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Not a valid UUID"}, w, 400)
		return
	}

	result, jobErr := GetJobForId(jobId, h.RedisClient)
	if jobErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not retrieve entry"}, w, 500)
		return
	}

	bodyContent := map[string]interface{}{"status": "ok", "entry": result}

	helpers.WriteJsonContent(bodyContent, w, 200)
}

type CreateJobHandler struct {
	RedisClient *redis.Client
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

	newEntry := NewJobEntry(rq.SettingsId)
	jobErr := PutJob(&newEntry, h.RedisClient)
	if jobErr != nil {
		log.Print("Could not save new job: ", jobErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not save record"}, w, 500)
		return
	}

	w.WriteHeader(201)
}

type ListJobHandler struct {
	RedisClient *redis.Client
}

func (h ListJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	response := h.RedisClient.Scan(0, "mediaflipper:job:*", 100)
	keys, _, err := response.Result()

	if err != nil {
		log.Printf("Could not list keys: ", err)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not iterate"}, w, 500)
		return
	}

	pipe := h.RedisClient.Pipeline()

	var cmds []*redis.StringStringMapCmd

	for _, k := range keys {
		cmds = append(cmds, pipe.HGetAll(k))
	}

	_, getErr := pipe.Exec()
	if getErr != nil {
		log.Printf("Could not retrieve data: ", err)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not retrieve data"}, w, 500)
		return
	}

	result := make([]*JobEntry, 0)

	for _, cmd := range cmds {
		content, getErr := cmd.Result()
		if getErr != nil {
			log.Printf("Could not retrieve data: ", getErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Could not retrieve data"}, w, 500)
			return
		}

		newEntry, marErr := JobEntryFromMap(content)
		if marErr != nil {
			log.Printf("offending data is %s", content)
			log.Printf("Could not unmarshal datastore content into job object for %s: %s", cmd.Args()[0], *marErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "Datastore data is not valid"}, w, 500)
			return
		}

		result = append(result, newEntry)
	}

	helpers.WriteJsonContent(map[string]interface{}{"status": "ok", "entries": result}, w, 200)
}

type StatusJobHandler struct {
	k8client *kubernetes.Clientset
}

func (h StatusJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)

	jobIdString := requestUrl.Query().Get("jobId")
	jobId, uuidErr := uuid.Parse(jobIdString)
	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Not a valid UUID"}, w, 400)
		return
	}

	jobResults, k8err := jobrunner.FindRunnerFor(jobId, h.k8client)
	if k8err != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "Could not retrieve job data from cluster",
		}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"entries": *jobResults,
	}, w, 200)
}
