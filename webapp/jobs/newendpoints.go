package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/models"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type JobsEndpoints struct {
	GetHandler    GetJobHandler
	CreateHandler CreateJobHandler
	ListHandler   ListJobHandler
}

func NewJobsEndpoints(redisClient *redis.Client, k8client *kubernetes.Clientset, jobTemplateMgr *models.JobTemplateManager) JobsEndpoints {
	return JobsEndpoints{
		GetHandler:    GetJobHandler{redisClient},
		CreateHandler: CreateJobHandler{redisClient, k8client, jobTemplateMgr},
		ListHandler:   ListJobHandler{redisClient},
	}
}

func (e JobsEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/get", e.GetHandler)
	http.Handle(baseUrlPath+"/new", e.CreateHandler)
	http.Handle(baseUrlPath+"", e.ListHandler)
}
