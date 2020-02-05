package jobs

import "github.com/google/uuid"

type JobRequest struct {
	JobTemplateId uuid.UUID `json:"jobTemplateId"`
}
