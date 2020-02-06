package analysis

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	models2 "github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
	"net/url"
	"strings"
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

	content, err := models2.GetFileFormat(jobId, h.redisClient)
	if err != nil {
		if strings.Contains(err.Error(), "redis: nil") { //FIXME: there must be a better way of doing this??
			helpers.WriteJsonContent(helpers.GenericErrorResponse{
				Status: "not_found",
				Detail: "no analysis data for that job id",
			}, w, 404)
		} else {
			log.Print("Could not read datastore content: ", err)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{
				Status: "db_error",
				Detail: "Could not read content from datastore",
			}, w, 500)
		}
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status": "ok",
		"entry":  content,
	}, w, 200)
}
