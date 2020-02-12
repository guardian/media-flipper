package bulkprocessor

import "github.com/google/uuid"

type BulkListUpdate struct {
	TemplateId uuid.UUID `json:"templateId"`
	NickName   string    `json:"nickName"`
}
