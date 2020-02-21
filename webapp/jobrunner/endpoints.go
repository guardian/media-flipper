package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type JobRunnerEndpoints struct {
	QueueStats   QueueStatsHandler
	PurgeHandler PurgeHandler
	EnqueueBulk  BulkEnqueueHandler
}

func NewJobRunnerEndpoints(redisClient *redis.Client, templateMgr *models.JobTemplateManager, runner *JobRunner) JobRunnerEndpoints {
	return JobRunnerEndpoints{
		QueueStats:   QueueStatsHandler{redisClient: redisClient},
		PurgeHandler: PurgeHandler{redisClient: redisClient},
		EnqueueBulk:  BulkEnqueueHandler{redisClient: redisClient, templateManager: templateMgr, runner: runner},
	}
}

func (e JobRunnerEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/queuestats", e.QueueStats)
	http.Handle(baseUrl+"/purge", e.PurgeHandler)
	http.Handle(baseUrl+"/enqueue", e.EnqueueBulk)
}
