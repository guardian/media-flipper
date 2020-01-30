package files

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
	"net/url"
)

type GetFileInfo struct {
	redisClient *redis.Client
}

/**
retrieve file metadata for the given file id
*/
func (h GetFileInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	uuidText := requestUrl.Query().Get("forId")
	fileId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "Invalid file ID",
		}, w, 400)
		return
	}

	entry, err := models.FileEntryForId(fileId, h.redisClient)
	if err != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "could not retrieve record from database",
		}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status": "ok",
		"entry":  entry,
	}, w, 200)
}
