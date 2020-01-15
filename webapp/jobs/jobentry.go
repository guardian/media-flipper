package jobs

import "github.com/google/uuid"

type JobStatus int

const (
	JOB_PENDING JobStatus = iota
	JOB_STARTED
	JOB_COMPLETED
	JOB_FAILED
)

func (j JobStatus) isFailure() bool {
	return j == JOB_FAILED
}

func (j JobStatus) isCompleted() bool {
	return j == JOB_FAILED || j == JOB_COMPLETED
}

type JobEntry struct {
	JobId      uuid.UUID `json:"containerId"`
	MediaFile  string    `json:"mediaFile"`
	SettingsId uuid.UUID `json:"settingsId"`
	Status     JobStatus `json:"jobStatus"`
}
