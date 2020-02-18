package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
)

type RemoveNonTranscodableHandler struct {
	redisClient *redis.Client
	dao         BulkListDAO
}

/**
callback function that removes items from the bulk if there is no transcode setting present for them
*/
func removeNTProcessor(itemsChan chan BulkItem, errChan chan error, outputChan chan error, batch BulkList, redisClient redis.Cmdable) {
	hasImageSetting := batch.GetImageTemplateId() != uuid.UUID{}
	hasVideoSetting := batch.GetVideoTemplateId() != uuid.UUID{}
	hasAudioSetting := batch.GetAudioTemplateId() != uuid.UUID{}

	for {
		select {
		case item := <-itemsChan:
			if item == nil {
				log.Printf("Remove non-transcodable run completed")
				batch.ClearActionRunning(REMOVE_NONTRANSCODABLE_FILES, redisClient)
				outputChan <- nil
				return
			}
			var removeErr error

			switch item.GetItemType() {
			case helpers.ITEM_TYPE_OTHER: //we never know what to do with these
				removeErr = batch.RemoveRecord(item, redisClient)
			case helpers.ITEM_TYPE_IMAGE: //remove image if there is no image preset
				if !hasImageSetting {
					removeErr = batch.RemoveRecord(item, redisClient)
				}
			case helpers.ITEM_TYPE_VIDEO:
				if !hasVideoSetting {
					removeErr = batch.RemoveRecord(item, redisClient)
				}
			case helpers.ITEM_TYPE_AUDIO:
				if !hasAudioSetting {
					removeErr = batch.RemoveRecord(item, redisClient)
				}
			}

			if removeErr != nil {
				log.Printf("WARNING: Could not remove item %s: %s", item.GetId(), removeErr)
			}
		case err := <-errChan:
			log.Printf("ERROR: Could not iterate all items: %s", err)
			batch.ClearActionRunning(REMOVE_NONTRANSCODABLE_FILES, redisClient)
			outputChan <- err
		}
	}
}

func (h RemoveNonTranscodableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	completionChan := RunAsyncActionForBatch(h.dao, *batchId, REMOVE_NONTRANSCODABLE_FILES, h.redisClient, removeNTProcessor)

	if syncMode {
		err := <-completionChan
		if err != nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{
				Status: "error",
				Detail: "could not complete run, see server logs for details",
			}, w, 500)
			return
		} else {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "action completed"}, w, 200)
		}
	} else {
		go func() {
			<-completionChan //ensure that the goroutine can terminate easily
		}()
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "action started"}, w, 200)
	}
}
