package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"net/http"
)

type GetHandler struct {
	redisClient *redis.Client
}

func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	_, bulkId, parseErr := helpers.GetForId(r.RequestURI)
	if parseErr != nil {
		helpers.WriteJsonContent(parseErr, w, 400)
		return
	}

	listPtr, getErr := BulkListForId(*bulkId, h.redisClient)
	if getErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not retrieve bulk list"}, w, 500)
		return
	}

	itemStats, getStatsErr := listPtr.CountForAllStates(h.redisClient)
	if getStatsErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not retvieve bulk stats"}, w, 500)
		return
	}

	rsp := BulkListGetResponse{
		BulkListId:     listPtr.GetId(),
		CreationTime:   listPtr.GetCreationTime(),
		PendingCount:   itemStats[ITEM_STATE_PENDING],
		ActiveCount:    itemStats[ITEM_STATE_ACTIVE],
		CompletedCount: itemStats[ITEM_STATE_COMPLETED],
		ErrorCount:     itemStats[ITEM_STATE_FAILED],
	}
	helpers.WriteJsonContent(&rsp, w, 200)
}
