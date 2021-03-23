package main

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
)

func AsyncFilterBulkId(inputCh chan *models.JobContainer, bulkId *uuid.UUID, queueSize int) chan *models.JobContainer {
	outputCh := make(chan *models.JobContainer, queueSize)

	go func() {
		for {
			rec := <- inputCh
			if rec==nil {
				outputCh <- nil
				return
			}
			if bulkId==nil {
				outputCh <- rec
			} else {
				if rec.AssociatedBulk.List==*bulkId {
					outputCh <- rec
				}
			}
		}
	}()
	return outputCh
}
