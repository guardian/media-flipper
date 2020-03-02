package bulkprocessor

import (
	"github.com/google/uuid"
	"time"
)

type BulkListGetResponse struct {
	BulkListId      uuid.UUID `json:"bulkListId"`
	NickName        string    `json:"nickName"`
	VideoTemplateId uuid.UUID `json:"videoTemplateId"`
	AudioTemplateId uuid.UUID `json:"audioTemplateId"`
	ImageTemplateId uuid.UUID `json:"imageTemplateId"`
	CreationTime    time.Time `json:"creationTime"`
	PendingCount    int64     `json:"pendingCount"`
	ActiveCount     int64     `json:"activeCount"`
	CompletedCount  int64     `json:"completedCount"`
	ErrorCount      int64     `json:"errorCount"`
	AbortedCount    int64     `json:"abortedCount"`
	NonQueuedCount  int64     `json:"nonQueuedCount"`
	RunningActions  []string  `json:"runningActions"`
}
