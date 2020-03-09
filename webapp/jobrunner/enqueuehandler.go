package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
)

type BulkEnqueueHandler struct {
	redisClient     *redis.Client
	templateManager *models.JobTemplateManager
	runner          *JobRunner
}

func (h BulkEnqueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	parsedUrl, batchId, urlErr := helpers.GetForId(r.RequestURI)

	if urlErr != nil {
		log.Printf("Could not parse out url: %s. Offending data was %s.", urlErr, r.RequestURI)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not parse out url"}, w, 500)
		return
	}

	syncMode := false
	if parsedUrl.Query().Get("sync") != "" {
		syncMode = true
	}

	batchList, getErr := bulk_models.BulkListForId(*batchId, h.redisClient)
	if getErr != nil {
		log.Printf("ERROR: Could not get bulk list for %s: %s", *batchId, getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "could not get batch list",
		}, w, 500)
		return
	}

	batchListImplPtr, isok := batchList.(*bulk_models.BulkListImpl)
	if !isok {
		log.Printf("ERROR: BulkEnqueueHandler does not yet support mocked BulkList")
		return
	}

	var specificItemUuid *uuid.UUID
	specificItemIdString := parsedUrl.Query().Get("forItem")
	if specificItemIdString == "" {
		specificItemUuid = nil
	} else {
		var parseErr error
		uid, parseErr := uuid.Parse(specificItemIdString)

		if parseErr != nil {
			log.Printf("BulkEnqueueHandler ERROR invalid forItem key: '%s' (%s)", specificItemIdString, parseErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "invalid forItem key"}, w, 500)
			return
		}
		specificItemUuid = &uid
	}

	var forState bulk_models.BulkItemState
	forStateString := parsedUrl.Query().Get("forState")
	if forStateString == "" && specificItemUuid == nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_arguments", "you must specify either forState or forItem"}, w, 400)
		return
	}

	forState = bulk_models.ItemStateFromString(forStateString)

	completionChan := h.runner.EnqueueContentsAsync(h.redisClient, h.templateManager, batchListImplPtr, specificItemUuid, forState, nil)

	if syncMode {
		log.Printf("INFO: BulkEnqueueHandler running in sync mode, waiting for completion...")
		gotErr := <-completionChan
		if gotErr == nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "enqueued batch"}, w, 201)
			return
		} else {
			log.Printf("ERROR: BulkEnqueueHandler async operation failed: %s", gotErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", gotErr.Error()}, w, 500)
		}
	} else {
		//ensure that the channel is read and can shut down
		go func() {
			gotErr := <-completionChan
			if gotErr != nil {
				log.Printf("ERROR: BulkEnqueueHandler async operation failed, can't inform client: %s", gotErr)
			}
		}()
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "enqueue operation running in background"}, w, 200)
	}
}
