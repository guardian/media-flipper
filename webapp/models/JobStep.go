package models

import (
	"github.com/google/uuid"
)

type JobStep interface {
	StepId() uuid.UUID
	ContainerId() uuid.UUID
	Status() JobStatus
	WithNewStatus(newStatus JobStatus, errorMsg *string) JobStep
	OutputId() *uuid.UUID
	OutputData() interface{}
	TimeTaken() float64
	ErrorMessage() string
	RunnerDesc() *JobRunnerDesc
	WithNewMediaFile(newMediaFile string) JobStep
}

func MapFromJobstep(from JobStep) map[string]interface{} {
	return map[string]interface{}{
		"stepId":       from.StepId(),
		"jobContainer": from.ContainerId(),
		"status":       from.Status(),
		"timeTaken":    from.TimeTaken(),
		"error":        from.ErrorMessage(),
	}
}
