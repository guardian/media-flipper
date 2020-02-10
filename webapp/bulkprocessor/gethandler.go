package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"net/http"
	"time"
)

type GetHandler struct {
	redisClient *redis.Client
}

type BulkListGetResponse struct {
	BulkListId     uuid.UUID `json:"bulkListId"`
	CreationTime   time.Time `json:"creationTime"`
	PendingCount   int64     `json:"pendingCount"`
	ActiveCount    int64     `json:"activeCount"`
	CompletedCount int64     `json:"completedCount"`
	ErrorCount     int64     `json:"errorCount"`
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
