package models

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
)

type JobStep interface {
	StepId() uuid.UUID
	ContainerId() uuid.UUID
	Status() JobStatus
	OutputPath() string
	OutputData() interface{}
	TimeTaken() float64
	ErrorMessage() string
	RunnerDesc() *JobRunnerDesc
	Store(redisClient *redis.Client) error
}

func MapFromJobstep(from JobStep) map[string]interface{} {
	return map[string]interface{}{
		"status":    from.Status(),
		"timeTaken": from.TimeTaken(),
		"error":     from.ErrorMessage(),
	}
}
