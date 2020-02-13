package bulkprocessor

import (
	"github.com/google/uuid"
	"time"
)

type BulkListGetResponse struct {
	BulkListId     uuid.UUID `json:"bulkListId"`
	NickName       string    `json:"nickName"`
	TemplateId     uuid.UUID `json:"templateId"`
	CreationTime   time.Time `json:"creationTime"`
	PendingCount   int64     `json:"pendingCount"`
	ActiveCount    int64     `json:"activeCount"`
	CompletedCount int64     `json:"completedCount"`
	ErrorCount     int64     `json:"errorCount"`
	RunningActions []string  `json:"runningActions"`
}
