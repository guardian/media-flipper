package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type JobRunnerEndpoints struct {
	QueueStats   QueueStatsHandler
	PurgeHandler PurgeHandler
}

func NewJobRunnerEndpoints(redisClient *redis.Client) JobRunnerEndpoints {
	return JobRunnerEndpoints{
		QueueStats:   QueueStatsHandler{redisClient: redisClient},
		PurgeHandler: PurgeHandler{redisClient: redisClient},
	}
}

func (e JobRunnerEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/queuestats", e.QueueStats)
	http.Handle(baseUrl+"/purge", e.PurgeHandler)
}
