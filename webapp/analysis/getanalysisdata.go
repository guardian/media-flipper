package analysis

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
	"net/url"
)

type GetData struct {
	redisClient *redis.Client
}

func (h GetData) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "expected GET",
		}, w, 405)
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	uuidText := requestUrl.Query().Get("forId")
	jobId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "Invalid file ID",
		}, w, 400)
		return
	}

	content, err := models.GetFileFormat(jobId, h.redisClient)
	if err != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "Could not read content from datastore",
		}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status": "ok",
		"entry":  content,
	}, w, 200)
}
