package jobrunner

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
)

type FailPendingHandler struct {
	redisClient *redis.Client
	runner      JobRunnerIF
}

/**
async handler that performs "remove from queue" on any jobs associated with the batch items that come through.
relays the items that come in to the output channel and stops on an error
arguments:
- recordsChan - channel that should yield BulkItem objects to process
- upstreamErrChan - channel that an upstream can use to indicate an error. If a non-nil value is received then the async processor forwards the error and shuts down.
returns:
- a channel that forwards on each record once it has been processed
- a channel that forwards on any error
*/
func (h FailPendingHandler) batchRemoveAsync(recordsChan chan bulk_models.BulkItem, upstreamErrChan chan error) (chan bulk_models.BulkItem, chan error) {
	recordsOutChan := make(chan bulk_models.BulkItem, 10)
	errorOutChan := make(chan error, 10)

	go func() {
		errChanTerminated := false

		for {
			select {
			case record := <-recordsChan:
				if record == nil {
					log.Printf("INFO FailPendingHandler.batchRemoveAsync reached end of list")
					recordsOutChan <- nil
					if !errChanTerminated { //if we have not yet received a null from the error channel, then don't leak memory; wait for it here asynchronously
						go func() {
							<-upstreamErrChan
							errorOutChan <- nil
						}()
					}
					return
				}
				jobContainer, getErr := models.JobContainerForBulkItem(record.GetId(), h.redisClient)
				if getErr != nil {
					log.Printf("WARNING FailPendingHandler.batchRemoveAsync could not retrieve job container for %s: %s", record.GetId(), getErr)
				} else {
					if jobContainer != nil {
						//take a blanket approach. try to remove any possible job from the running queue.
						for _, c := range jobContainer {
							removeErr := h.runner.RemoveJob(&c)
							if removeErr != nil {
								log.Printf("ERROR FailPendingHandler.batchRemoveAsync could not remove %s from the running queue: %s", c.Id, removeErr)
							}
						}
					}
				}
				recordsOutChan <- record
			case err := <-upstreamErrChan:
				if err == nil {
					errChanTerminated = true
				} else {
					log.Printf("ERROR FailPendingHandler.batchRemoveAsync got upstream error: %s", err)
					return //we expect nothing more from the recordsChan now
				}
			}
		}
	}()

	return recordsOutChan, errorOutChan
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

	bulkList, lookupErr := bulk_models.BulkListForId(*forId, h.redisClient)
	if lookupErr != nil {
		log.Printf("ERROR FailPendingHandler could not look up bulk list for %s: %s", *forId, lookupErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not look up bulk list"}, w, 500)
		return
	}

	bulkItemStream, filterErrStream := bulkList.FilterRecordsByStateAsync(bulk_models.ITEM_STATE_PENDING, h.redisClient)
	removedStream, removeErrStream := h.batchRemoveAsync(bulkItemStream, filterErrStream)
	updateErrStream := asyncUpdateItemStatus(removedStream, bulk_models.ITEM_STATE_FAILED, bulkList, 100, h.redisClient)

	go func() {
		filterErr := <-removeErrStream
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
