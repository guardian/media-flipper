package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
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

	runningActions, runningActionsErr := listPtr.GetActionsRunning(h.redisClient)
	if runningActionsErr != nil {
		log.Printf("ERROR: could not list running actions: %s", runningActionsErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not retvieve running actions"}, w, 500)
		return
	}

	runningActionsStrings := make([]string, len(runningActions))
	for i, a := range runningActions {
		runningActionsStrings[i] = string(a)
	}

	rsp := BulkListGetResponse{
		BulkListId:     listPtr.GetId(),
		CreationTime:   listPtr.GetCreationTime(),
		NickName:       listPtr.GetNickName(),
		TemplateId:     listPtr.GetTemplateId(),
		PendingCount:   itemStats[ITEM_STATE_PENDING],
		ActiveCount:    itemStats[ITEM_STATE_ACTIVE],
		CompletedCount: itemStats[ITEM_STATE_COMPLETED],
		ErrorCount:     itemStats[ITEM_STATE_FAILED],
		RunningActions: runningActionsStrings,
	}
	helpers.WriteJsonContent(&rsp, w, 200)
}
