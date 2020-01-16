package jobs

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
)

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

func (j JobEntry) toMap() map[string]string {
	return map[string]string{
		"jobId":      j.JobId.String(),
		"mediaFile":  j.MediaFile,
		"settingsId": j.SettingsId.String(),
		"status":     fmt.Sprintf("%d", j.Status),
	}
}

func JobEntryFromMap(fromData map[string]string) (*JobEntry, *error) {
	jobId, jobIdErr := uuid.Parse(fromData["jobId"])
	if jobIdErr != nil {
		return nil, &jobIdErr
	}
	settingsId, settingsIdErr := uuid.Parse(fromData["settingsId"])
	if settingsIdErr != nil {
		return nil, &settingsIdErr
	}

	statusNum, statusNumErr := strconv.Atoi(fromData["status"])
	if statusNumErr != nil {
		return nil, &statusNumErr
	}

	return &JobEntry{
		jobId,
		fromData["mediaFile"],
		settingsId,
		JobStatus(statusNum),
	}, nil
}

func NewJobEntry(settingsId uuid.UUID) JobEntry {
	return JobEntry{
		JobId:      uuid.New(),
		MediaFile:  "",
		SettingsId: settingsId,
		Status:     JOB_PENDING,
	}
}
