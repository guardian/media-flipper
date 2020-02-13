package bulkprocessor

import "github.com/google/uuid"

type BulkListUpdate struct {
	VideoTemplateId uuid.UUID `json:"videoTemplateId"`
	AudioTemplateId uuid.UUID `json:"audioTemplateId"`
	ImageTemplateId uuid.UUID `json:"imageTemplateId"`
	NickName        string    `json:"nickName"`
}
