package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type JobRunnerEndpoints struct {
	QueueStats    QueueStatsHandler
	PurgeHandler  PurgeHandler
	EnqueueBulk   BulkEnqueueHandler
	ManualCleanup ManualCleanupHandler
	FailPending   FailPendingHandler
}

func NewJobRunnerEndpoints(redisClient *redis.Client, templateMgr *models.JobTemplateManager, runner *JobRunner, clientset *kubernetes.Clientset) JobRunnerEndpoints {
	return JobRunnerEndpoints{
		QueueStats:    QueueStatsHandler{redisClient: redisClient},
		PurgeHandler:  PurgeHandler{redisClient: redisClient},
		EnqueueBulk:   BulkEnqueueHandler{redisClient: redisClient, templateManager: templateMgr, runner: runner},
		ManualCleanup: ManualCleanupHandler{redisClient: redisClient, k8clientset: clientset},
		FailPending:   FailPendingHandler{redisClient: redisClient, runner: runner},
	}
}

func (e JobRunnerEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/queuestats", e.QueueStats)
	http.Handle(baseUrl+"/purge", e.PurgeHandler)
	http.Handle(baseUrl+"/enqueue", e.EnqueueBulk)
	http.Handle(baseUrl+"/cleanup", e.ManualCleanup)
	http.Handle(baseUrl+"/failpending", e.FailPending)
}
