package files

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type StreamFile struct {
	redisClient *redis.Client
}

/**
stream out the contents of the given file
*/
func (h StreamFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	reader, openErr := os.OpenFile(entry.ServerPath, os.O_RDONLY, 0755)
	if openErr != nil {
		log.Printf("Could not open %s for %s: %s", entry.ServerPath, entry.Id, openErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "could not open file server-side",
		}, w, 500)
	}

	w.Header().Add("Content-Type", entry.MimeType)
	w.Header().Add("Content-Length", strconv.FormatInt(entry.Size, 10))
	w.WriteHeader(200)
	bytesCopied, copyErr := io.Copy(w, reader)
	if copyErr != nil {
		log.Printf("Could not stream out %s for %s: %s", entry.ServerPath, entry.Id, copyErr)
	}
	if bytesCopied < entry.Size {
		log.Printf("Stream of %s for %s truncated, expected %d bytes but streamed %d", entry.ServerPath, entry.Id, entry.Size, bytesCopied)
	}
}
