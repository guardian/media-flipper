package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
)

type DeleteHandler struct {
	redisClient *redis.Client
}

func processItems(itemsChan chan BulkItem, errChan chan error, completionChan chan error, redisClient redis.Cmdable) {
	pipe := redisClient.Pipeline()
	defer pipe.Close()
	for {
		select {
		case item := <-itemsChan:
			if item == nil {
				_, execErr := pipe.Exec()
				if execErr != nil {
					log.Printf("ERROR: could not perform deletion: %s", execErr)
					completionChan <- execErr
					return
				}
				log.Printf("All items deleted")
				completionChan <- nil
				return
			}
			deleteErr := item.Delete(pipe)
			if deleteErr != nil {
				log.Printf("Could not delete item %s from %s: %s", item.GetId(), item.GetBulkId(), deleteErr)
			}
		case err := <-errChan:
			log.Printf("could not iterate all items: %s", err)
			_, execErr := pipe.Exec()
			if execErr != nil {
				log.Printf("ERROR: could not perform deletion: %s", execErr)
				completionChan <- execErr
				return
			}
			completionChan <- err
			return
		}
	}
}

func (h DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "DELETE") {
		return
	}

	parsedUrl, batchId, urlErr := helpers.GetForId(r.RequestURI)
	syncMode := false
	if parsedUrl.Query().Get("sync") != "" {
		syncMode = true
	}

	if urlErr != nil {
		helpers.WriteJsonContent(urlErr, w, 500)
		return
	}

	bulkList, listGetErr := BulkListForId(*batchId, h.redisClient)
	if listGetErr != nil {
		log.Printf("ERROR: could not retrieve batch list for %s: %s", *batchId, listGetErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not retrieve batch list"}, w, 500)
		return
	}

	/**
	asynchronously retrieve and individually delete items
	*/
	itemsCompletedChan := make(chan error, 2)
	itemsChan, errChan := bulkList.GetAllRecordsAsync(h.redisClient)
	go processItems(itemsChan, errChan, itemsCompletedChan, h.redisClient)

	finalCompletedChan := make(chan error)
	go func() {
		err := <-itemsCompletedChan
		if err != nil {
			finalCompletedChan <- err
			return
		}

		bulkList.Delete(h.redisClient)
	}()

	if syncMode { //wait for operations to complete only if we have been asked to
		<-itemsCompletedChan
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "deletion completed"}, w, 200)
	} else {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "deletion started"}, w, 200)
	}

}
