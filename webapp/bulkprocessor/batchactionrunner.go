package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"log"
)

/**
generic function that runs a callback against every member of the given batch
returns a channel that completes with nil if no error occurred, or an error object describing what went wrong
*/
func RunAsyncActionForBatch(dao bulk_models.BulkListDAO, batchId uuid.UUID, actionId bulk_models.BulkListAction, redisClient redis.Cmdable, asyncCallback func(chan bulk_models.BulkItem, chan error, chan error, bulk_models.BulkList, redis.Cmdable)) chan error {
	competionChan := make(chan error)

	go func() {
		batch, getErr := dao.BulkListForId(batchId, redisClient)
		//retrieve the batch-list and write a flag to say that the job is starting up
		if getErr != nil {
			log.Printf("could not get batch list for %s: %s", batchId, getErr)
			competionChan <- getErr
			return
		}

		setErr := batch.SetActionRunning(actionId, redisClient)
		if setErr != nil {
			log.Printf("Could not set action running flag: %s", setErr)
			competionChan <- setErr
			return
		}

		processorCompleted := make(chan error)

		//start item retrieve and pass them through to async callback via a channel
		itemsChan, errChan := batch.GetAllRecordsAsync(redisClient)

		go asyncCallback(itemsChan, errChan, processorCompleted, batch, redisClient)

		//this function will wait until the async callback has finished and then clear the action running flag
		go func() {
			maybeProcessorErr := <-processorCompleted
			batch.ClearActionRunning(actionId, redisClient)
			competionChan <- maybeProcessorErr
		}()
	}()

	return competionChan
}
