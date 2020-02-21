package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
)

type QueueStatsHandler struct {
	redisClient *redis.Client
}

func (h QueueStatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	result, getErr := models.AllQueuesLength(h.redisClient)
	if getErr != nil {
		log.Printf("ERROR: QueueStatsHandler could not get queue stats: %s", getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", getErr.Error()}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{"status": "ok", "queues": result}, w, 200)
	return
}
