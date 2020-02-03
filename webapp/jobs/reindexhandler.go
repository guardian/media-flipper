package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"log"
	"net/http"
)

type ReindexHandler struct {
	redisClient *redis.Client
}

func (h ReindexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "PUT") {
		return
	}

	err := models.ReIndexJobContainers(h.redisClient)
	if err != nil {
		log.Printf("Reindex operation failed!")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: err.Error(),
		}, w, 500)
	} else {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "ok",
			Detail: "reindex completed",
		}, w, 200)
	}

}
