package jobs

import (
	"github.com/go-redis/redis/v7"
	models2 "github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type JobsEndpoints struct {
	GetHandler     GetJobHandler
	CreateHandler  CreateJobHandler
	ListHandler    ListJobHandler
	ReindexHandler ReindexHandler
}

func NewJobsEndpoints(redisClient *redis.Client, k8client *kubernetes.Clientset, jobTemplateMgr *models2.JobTemplateManager) JobsEndpoints {
	return JobsEndpoints{
		GetHandler:     GetJobHandler{redisClient},
		CreateHandler:  CreateJobHandler{redisClient, k8client, jobTemplateMgr},
		ListHandler:    ListJobHandler{redisClient},
		ReindexHandler: ReindexHandler{redisClient},
	}
}

func (e JobsEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/get", e.GetHandler)
	http.Handle(baseUrlPath+"/new", e.CreateHandler)
	http.Handle(baseUrlPath+"", e.ListHandler)
	http.Handle(baseUrlPath+"/reindex", e.ReindexHandler)
}
