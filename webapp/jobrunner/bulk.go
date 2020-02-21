package jobrunner

import (
	"errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"github.com/guardian/mediaflipper/webapp/bulkprocessor"
	"log"
)

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
							log.Printf("ERROR: could not commit all records: %s", putErr)
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
				updatedRecord.Store(currentPipeline)
				l.ReindexRecord(updatedRecord, rec, currentPipeline)

				pendingCount++
				if pendingCount >= commitEvery {
					_, putErr := currentPipeline.Exec()
					if putErr != nil {
						log.Printf("ERROR: could not commit all records: %s", putErr)
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
*/
func (runner *JobRunner) EnqueueContentsAsync(redisClient redis.Cmdable, templateManager models.TemplateManagerIF, l *bulkprocessor.BulkListImpl) chan error {
	rtnChan := make(chan error, 10)

	l.SetActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
	l.Store(redisClient)

	resultsChan, errChan := l.GetAllRecordsAsync(redisClient)

	updateChan := make(chan bulkprocessor.BulkItem, 10)
	storeErrChan := asyncUpdateItemStatus(updateChan, bulkprocessor.ITEM_STATE_PENDING, l, 50, redisClient)
	go func() {
		<-storeErrChan
	}()

	go func() {
		for {
			select {
			case rec := <-resultsChan:
				if rec == nil {
					log.Printf("Completed enqueueing contents")
					l.ClearActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
					l.Store(redisClient)
					updateChan <- nil
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
					log.Printf("EnqueueContentsAsync: job before writing is %s", spew.Sdump(job))
					storErr := job.Store(redisClient)
					if storErr != nil {
						log.Printf("ERROR: Could not store new job %s for bulk item %s: %s", job.Id, rec.GetId(), storErr)
					} else {
						addErr := runner.AddJob(job)
						updateChan <- rec //bulk-store the updated records
						if addErr != nil {
							log.Printf("ERROR: Could not add created job to jobrunner: %s", addErr)
						}
					}
				} else {
					log.Printf("ERROR: could not build job for %s %s: %s", rec.GetItemType(), rec.GetSourcePath(), buildErr)
				}
			case err := <-errChan:
				l.ClearActionRunning(bulkprocessor.JOBS_QUEUEING, redisClient)
				l.Store(redisClient)
				rtnChan <- err
				return
			}
		}
	}()

	return rtnChan
}
