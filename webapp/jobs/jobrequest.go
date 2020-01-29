package jobs

import "github.com/google/uuid"

type JobRequest struct {
	SettingsId    uuid.UUID `json:"settingsId"`
	JobTemplateId uuid.UUID `json:"jobTemplateId"`
}
