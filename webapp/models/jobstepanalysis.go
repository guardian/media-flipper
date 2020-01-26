package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

type JobStepAnalysis struct {
	JobStepId              uuid.UUID      `json:"id"`
	JobContainerId         uuid.UUID      `json:"JobContainerId"`
	ContainerData          *JobRunnerDesc `json:"containerData"`
	StatusValue            JobStatus      `json:"jobStepStatus"`
	Result                 AnalysisResult `json:"analysisResult"`
	MediaFile              string         `json:"mediaFile"`
	KubernetesTemplateFile string         `json:"templateFile"`
}

func (j JobStepAnalysis) StepId() uuid.UUID {
	return j.JobStepId
}

func (j JobStepAnalysis) ContainerId() uuid.UUID {
	return j.JobContainerId
}

func (j JobStepAnalysis) Status() JobStatus {
	return j.StatusValue
}

func (j JobStepAnalysis) OutputPath() string {
	return ""
}

func (j JobStepAnalysis) OutputData() interface{} {
	return j.Result
}

func (j JobStepAnalysis) RunnerDesc() *JobRunnerDesc {
	return j.ContainerData
}

func (j JobStepAnalysis) TimeTaken() float64 {
	return -1
}

func (j JobStepAnalysis) ErrorMessage() string {
	return ""
}

func (j JobStepAnalysis) Store(redisClient *redis.Client) error {
	dbKey := fmt.Sprintf("mediaflipper:JobStepAnalysis:%s", j.JobStepId.String())
	content, marshalErr := json.Marshal(j)
	if marshalErr != nil {
		log.Printf("Could not marshal content for jobstep %s: %s", j.JobStepId.String(), marshalErr)
		return marshalErr
	}

	_, dbErr := redisClient.Set(dbKey, string(content), -1).Result()
	if dbErr != nil {
		log.Printf("Could not save key for jobstep %s: %s", j.JobStepId.String(), dbErr)
		return dbErr
	}
	return nil
}

func (j JobStepAnalysis) WithNewStatus(newStatus JobStatus) JobStep {
	j.StatusValue = newStatus
	return j
}

func LoadJobStepAnalysis(fromId uuid.UUID, redisClient *redis.Client) (*JobStepAnalysis, error) {
	dbKey := fmt.Sprintf("mediaflipper:JobStepAnalysis:%s", fromId.String())
	content, getErr := redisClient.Get(dbKey).Result()

	if getErr != nil {
		log.Printf("Could not load key for jobstep %s: %s", fromId.String(), getErr)
		return nil, getErr
	}

	var rtn JobStepAnalysis
	marshalErr := json.Unmarshal([]byte(content), &rtn)
	if marshalErr != nil {
		log.Printf("Could not understand data for jobstep %s: %s", fromId.String(), marshalErr)
		log.Printf("Offending data was %s", content)
		return nil, marshalErr
	}

	return &rtn, nil
}