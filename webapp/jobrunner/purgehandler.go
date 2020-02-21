package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
	"net/url"
)

type PurgeHandler struct {
	redisClient *redis.Client
}

func isValidQueuename(name string) bool {
	for _, qn := range models.ALL_QUEUES {
		if name == string(qn) {
			return true
		}
	}
	return false
}

func (h PurgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "DELETE") {
		return
	}

	parsedurl, parseErr := url.Parse(r.RequestURI)
	if parseErr != nil {
		log.Printf("ERROR: PurgeHandler could not parse incoming uri '%s': %s", r.RequestURI, parseErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not parse url from framework"}, w, 500)
		return
	}

	queueName := parsedurl.Query().Get("queue")
	if queueName == "" || !isValidQueuename(queueName) {
		log.Printf("ERROR: provided queue name '%s' is not valid", queueName)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "invalid queue name"}, w, 400)
		return
	}

	purgErr := models.PurgeQueue(h.redisClient, models.QueueName(queueName))
	if purgErr != nil {
		log.Printf("ERROR: PurgeHandler could not purge %s: %s", queueName, purgErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", purgErr.Error()}, w, 500)
		return
	}

	helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "queue purged"}, w, 200)
}
