package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
)

type StatusSummaryHandler struct {
	redisClient *redis.Client
}

func (h StatusSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	summaryData, getErr := models.JobStatusSummary(h.redisClient)
	if getErr != nil {
		log.Printf("ERROR StatusSummaryHandler could not get job statuses: %s", getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", getErr.Error()}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status": "ok",
		"data": map[string]int64{
			"pending":   (*summaryData)[models.JOB_PENDING],
			"started":   (*summaryData)[models.JOB_STARTED],
			"completed": (*summaryData)[models.JOB_COMPLETED],
			"failed":    (*summaryData)[models.JOB_FAILED],
			"aborted":   (*summaryData)[models.JOB_ABORTED],
			"notqueued": (*summaryData)[models.JOB_NOT_QUEUED],
		},
	}, w, 200)

}
