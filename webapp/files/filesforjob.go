package files

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
	"net/url"
)

type ListByJob struct {
	redisClient *redis.Client
}

/**
return a list of file entries associated with the given job id
*/
func (h ListByJob) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	uuidText := requestUrl.Query().Get("jobId")
	jobId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "Invalid file ID",
		}, w, 400)
		return
	}

	results, err := models.FilesForJobContainer(jobId, h.redisClient)
	if err != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "could not retrieve files for container",
		}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"count":   len(*results),
		"entries": *results,
	}, w, 200)
}
