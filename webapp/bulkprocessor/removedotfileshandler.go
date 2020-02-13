package bulkprocessor

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type RemoveDotFiles struct {
	redisClient *redis.Client
}

func itemProcessor(itemsChan chan BulkItem, errChan chan error, outputChan chan error, batch BulkList, redisClient redis.Cmdable) {
	for {
		select {
		case item := <-itemsChan:
			if item == nil {
				log.Printf("Remove dotfiles run completed")
				batch.ClearActionRunning(REMOVE_SYSTEM_FILES, redisClient)
				outputChan <- nil
				return
			}
			sourceFile := filepath.Base(item.GetSourcePath())
			if strings.HasPrefix(sourceFile, ".") {
				log.Printf("DEBUG: Removing record for %s", sourceFile)
				removeErr := batch.RemoveRecord(item, redisClient)
				if removeErr != nil {
					log.Printf("WARNING: Could not remove item %s: %s", spew.Sdump(item), removeErr)
				}
			}
		case err := <-errChan:
			log.Printf("ERROR: Could not iterate all items: %s", err)
			batch.ClearActionRunning(REMOVE_SYSTEM_FILES, redisClient)
			outputChan <- err
		}
	}
}

func (h RemoveDotFiles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	batch, getErr := BulkListForId(*batchId, h.redisClient)
	if getErr != nil {
		log.Printf("could not get batch list for %s: %s", *batchId, getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not get batch list"}, w, 500)
		return
	}

	setErr := batch.SetActionRunning(REMOVE_SYSTEM_FILES, h.redisClient)
	if setErr != nil {
		log.Printf("Could not set action running flag: %s", setErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not set action-running flag"}, w, 500)
		return
	}

	completionChan := make(chan error)
	//tempTimer := time.NewTimer(3*time.Second)

	itemsChan, errChan := batch.GetAllRecordsAsync(h.redisClient)

	go itemProcessor(itemsChan, errChan, completionChan, batch, h.redisClient)

	//go func() {
	//	<- tempTimer.C
	//	batch.ClearActionRunning(REMOVE_SYSTEM_FILES, h.redisClient)
	//	completionChan  <- nil
	//}()

	if syncMode {
		<-completionChan
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "action completed"}, w, 200)
	} else {
		go func() {
			<-completionChan //ensure that the goroutine can terminate easily
		}()
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "action started"}, w, 200)
	}
}
