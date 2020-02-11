package bulkprocessor

import (
	"github.com/google/uuid"
	"time"
)

type BulkListGetResponse struct {
	BulkListId     uuid.UUID `json:"bulkListId"`
	CreationTime   time.Time `json:"creationTime"`
	PendingCount   int64     `json:"pendingCount"`
	ActiveCount    int64     `json:"activeCount"`
	CompletedCount int64     `json:"completedCount"`
	ErrorCount     int64     `json:"errorCount"`
}
