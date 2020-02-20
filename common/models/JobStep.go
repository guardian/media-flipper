package models

import (
	"github.com/go-redis/redis/v7"
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
	DeleteAssociatedItems(redisClient redis.Cmdable) []error
}
