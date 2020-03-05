package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/webapp/bulkprocessor"
	"log"
	"net/http"
)

type FailPendingHandler struct {
	redisClient *redis.Client
}

func (h FailPendingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "PUT") {
		return
	}

	_, forId, errResponse := helpers.GetForId(r.RequestURI)
	if errResponse != nil {
		helpers.WriteJsonContent(errResponse, w, 500)
		return
	}

	bulkList, lookupErr := bulkprocessor.BulkListForId(*forId, h.redisClient)
	if lookupErr != nil {
		log.Printf("ERROR FailPendingHandler could not look up bulk list for %s: %s", *forId, lookupErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not look up bulk list"}, w, 500)
		return
	}

	bulkItemStream, filterErrStream := bulkList.FilterRecordsByStateAsync(bulkprocessor.ITEM_STATE_PENDING, h.redisClient)
	updateErrStream := asyncUpdateItemStatus(bulkItemStream, bulkprocessor.ITEM_STATE_FAILED, bulkList, 100, h.redisClient)

	go func() {
		filterErr := <-filterErrStream
		if filterErr != nil {
			log.Printf("ERROR FailPendingHandler got an error from the filter operation: %s", filterErr)
		} else {
			log.Printf("DEBUG FailPendingHandler got end of filter stream")
		}
	}()

	updateErr := <-updateErrStream
	if updateErr != nil {
		log.Printf("ERROR FailPendingHandler could not complete updating records: %s", updateErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not update all records"}, w, 500)
	}
	helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "records updated"}, w, 200)
}
