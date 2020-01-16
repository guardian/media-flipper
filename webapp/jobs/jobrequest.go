package jobs

import "github.com/google/uuid"

type JobRequest struct {
	SettingsId uuid.UUID `json:"settingsId"`
}
