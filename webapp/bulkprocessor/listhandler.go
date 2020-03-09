package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type ListHandler struct {
	redisClient *redis.Client
}

func (h ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	startAtString := requestUrl.Query().Get("start")
	limitString := requestUrl.Query().Get("limit")

	var startAt int64
	if startAtString == "" {
		startAt = 0
	} else {
		var startParseErr error
		startAt, startParseErr = strconv.ParseInt(startAtString, 10, 64)
		if startParseErr != nil {
			log.Printf("WARNING: Could not get start number from %s", startAtString)
			startAt = 0
		}
	}

	var endAt int64
	if limitString == "" {
		endAt = startAt + 50
	} else {
		limit, limParseErr := strconv.ParseInt(limitString, 10, 64)
		if limParseErr != nil {
			log.Printf("WARNING: Could not get limit number from %s", limParseErr)
			endAt = startAt + 50
		} else {
			endAt = startAt + limit
		}
	}

	results, getErr := bulk_models.ScanBulkList(startAt, endAt, h.redisClient)
	if getErr != nil {
		log.Printf("ERROR: could not retrieve bulk lists: %s", getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", getErr.Error()}, w, 500)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"entries": results,
	}, w, 200)
}
