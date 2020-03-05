package jobrunner

import (
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"github.com/guardian/mediaflipper/webapp/bulkprocessor"
	"log"
	"time"
)

/**
set up an asynchronous update of the status of a bunch of items.
pass in a channel that yields instances of BulkItem, and their status will be updated to the requested value and they will
be written back to the datastore layer in bulks of `commitEvery` records
pass a nil to the `records` channel to signal termination; any outstanding records will be written and the writer will terminate.
*/
func asyncUpdateItemStatus(records chan bulkprocessor.BulkItem, newState bulkprocessor.BulkItemState, l bulkprocessor.BulkList, commitEvery int, client redis.Cmdable) chan error {
	errorChan := make(chan error)

	go func() {
		var currentPipeline redis.Pipeliner = nil
		var pendingCount int = 0

		for {
			select {
			case rec := <-records:
				if rec == nil {
					if currentPipeline != nil {
						log.Printf("asyncStoreItemsBulk: got the last item, committing %d pending records...", pendingCount)
						_, putErr := currentPipeline.Exec()
						if putErr != nil {
							log.Printf("ERROR: jobrunner/asyncUpdateItemStatus could not commit all records: %s", putErr)
						}
						currentPipeline.Close()
					}
					errorChan <- nil
					return
				}

				if currentPipeline == nil {
					log.Printf("asyncStoreItemsBulk: creating new pipeline")
					currentPipeline = client.Pipeline()
				}

				updatedRecord := rec.CopyWithNewState(newState)
				//since we are pipelining these will never return errors (execution is not happening immediately)
				//so it's pointless testing error codes here
				updatedRecord.Store(currentPipeline)
				l.ReindexRecord(updatedRecord, rec, currentPipeline)

				pendingCount++
				if pendingCount >= commitEvery {
					_, putErr := currentPipeline.Exec()
					if putErr != nil {
						log.Printf("ERROR: jobrunner/asyncUpdateItemStatus could not commit all records: %s", putErr)
					}
					currentPipeline.Close()
					currentPipeline = nil
					pendingCount = 0
				}
			}
		}
	}()

	return errorChan
}

/**
put every item onto the waiting queue asynchronously
returns a channel that yields either an error if the operation fails or nil if it is successful
arguments:
- redisClient     - instance of redis.Cmdable (normally a *redis.Client, should not be a pipeline)
- templateManager - instance of a TemplateManagerInterface, used for building the jobs
- l               - the bulk list to work from
- maybeSpecificID - allows retrying of a single item, pass a pointer to the items UUID and only this one will be actioned. Otherwise pass nil.
- byState         - the item state that should be enqueued
- testRunner      - pass in an alternative JobRunner, used for testing
*/
func (runner *JobRunner) EnqueueContentsAsync(redisClient redis.Cmdable, templateManager models.TemplateManagerIF,
	l *bulkprocessor.BulkListImpl, maybeSpecificID *uuid.UUID, byState bulkprocessor.BulkItemState, testRunner JobRunnerIF) chan error {
	rtnChan := make(chan error, 10)

	l.SetActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
	l.Store(redisClient)

	var resultsChan chan bulkprocessor.BulkItem
	var errChan chan error

	if maybeSpecificID == nil {
		resultsChan, errChan = l.FilterRecordsByStateAsync(byState, redisClient)
	} else {
		resultsChan, errChan = l.GetSpecificRecordAsync(*maybeSpecificID, redisClient)
	}

	updateChan := make(chan bulkprocessor.BulkItem, 10)
	storeErrChan := asyncUpdateItemStatus(updateChan, bulkprocessor.ITEM_STATE_PENDING, l, 50, redisClient)

	go func() {
		for {
			select {
			case rec := <-resultsChan:
				if rec == nil {
					log.Printf("INFO EnqueueContentsAsync completed enqueueing contents, waiting for async store to complete")
					updateChan <- nil
					//wait for up to 2 seconds for the store thread to indicate that it has completed
					tmr := time.NewTimer(2 * time.Second)
					select {
					case <-storeErrChan:
						tmr.Stop()
						break
					case <-tmr.C:
						log.Printf("ERROR: EnqueueContentsAsync timed out while waiting for async store to complete")
					}

					log.Printf("INFO EnqueueContentsAsync store completed, exiting")
					l.ClearActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
					l.Store(redisClient)
					rtnChan <- nil
					return
				}
				var job *models.JobContainer
				var buildErr error
				switch rec.GetItemType() {
				case helpers.ITEM_TYPE_VIDEO:
					job, buildErr = templateManager.NewJobContainer(l.VideoTemplateId, rec.GetItemType())
				case helpers.ITEM_TYPE_AUDIO:
					job, buildErr = templateManager.NewJobContainer(l.AudioTemplateId, rec.GetItemType())
				case helpers.ITEM_TYPE_IMAGE:
					job, buildErr = templateManager.NewJobContainer(l.ImageTemplateId, rec.GetItemType())
				case helpers.ITEM_TYPE_OTHER:
					buildErr = errors.New("WARNING: can't enqueue an item of TYPE_OTHER, don't know what to do with it")
				default:
					buildErr = errors.New("ERROR: item had no item type! this should not happen")
				}

				if buildErr == nil {
					job.SetMediaFile(rec.GetSourcePath())
					job.AssociatedBulk = &models.BulkAssociation{
						Item: rec.GetId(),
						List: l.GetId(),
					}
					storErr := job.Store(redisClient)
					if storErr != nil {
						log.Printf("ERROR: Could not store new job %s for bulk item %s: %s", job.Id, rec.GetId(), storErr)
					} else {
						var addErr error
						if testRunner != nil {
							addErr = testRunner.AddJob(job)
						} else {
							addErr = runner.AddJob(job)
						}
						updateChan <- rec //bulk-store the updated records
						if addErr != nil {
							log.Printf("ERROR: Could not add created job to jobrunner: %s", addErr)
						}
					}
				} else {
					log.Printf("ERROR: could not build job for %s %s: %s", rec.GetItemType(), rec.GetSourcePath(), buildErr)
				}
			case err := <-errChan:
				if err != nil {
					l.ClearActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
					l.Store(redisClient)
					rtnChan <- err
					return
				}
			}
		}
	}()

	return rtnChan
}
