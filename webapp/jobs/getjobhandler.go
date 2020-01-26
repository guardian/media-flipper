package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
	"net/url"
)

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

	result, jobErr := models.JobContainerForId(jobId, h.RedisClient)
	if jobErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "Could not retrieve entry"}, w, 500)
		return
	}

	bodyContent := map[string]interface{}{"status": "ok", "entry": result}

	helpers.WriteJsonContent(bodyContent, w, 200)
}
