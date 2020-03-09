package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/bulk_models"
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

	listPtr, getErr := bulk_models.BulkListForId(*bulkId, h.redisClient)
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
		BulkListId:      listPtr.GetId(),
		CreationTime:    listPtr.GetCreationTime(),
		NickName:        listPtr.GetNickName(),
		VideoTemplateId: listPtr.GetVideoTemplateId(),
		AudioTemplateId: listPtr.GetAudioTemplateId(),
		ImageTemplateId: listPtr.GetImageTemplateId(),
		PendingCount:    itemStats[bulk_models.ITEM_STATE_PENDING],
		ActiveCount:     itemStats[bulk_models.ITEM_STATE_ACTIVE],
		CompletedCount:  itemStats[bulk_models.ITEM_STATE_COMPLETED],
		ErrorCount:      itemStats[bulk_models.ITEM_STATE_FAILED],
		AbortedCount:    itemStats[bulk_models.ITEM_STATE_ABORTED],
		NonQueuedCount:  itemStats[bulk_models.ITEM_STATE_NOT_QUEUED],
		LostCount:       itemStats[bulk_models.ITEM_STATE_LOST],
		RunningActions:  runningActionsStrings,
	}
	helpers.WriteJsonContent(&rsp, w, 200)
}
