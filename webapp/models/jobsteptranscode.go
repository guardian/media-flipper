package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"time"
)

type JobStepTranscode struct {
	JobStepType            string         `json:"stepType"` //this field is vital so we can correctly unmarshal json data from the store
	JobStepId              uuid.UUID      `json:"id"`
	JobContainerId         uuid.UUID      `json:"jobContainerId"`
	ContainerData          *JobRunnerDesc `json:"containerData"`
	StatusValue            JobStatus      `json:"jobStepStatus"`
	LastError              string         `json:"errorMessage",struct:"errorMessage"`
	MediaFile              string         `json:"mediaFile",struct:"mediaFile"`
	ResultId               *uuid.UUID     `json:"transcodeResult"`
	TimeTakenValue         float64        `json:"timeTaken",struct:"timeTaken"`
	KubernetesTemplateFile string         `json:"templateFile"`
	StartTime              *time.Time     `json:"startTime",struct:"startTime"`
	EndTime                *time.Time     `json:"endTime",struct:"startTime"`
}

func (j JobStepTranscode) StepId() uuid.UUID {
	return j.JobStepId
}

func (j JobStepTranscode) ContainerId() uuid.UUID {
	return j.JobContainerId
}

func (j JobStepTranscode) Status() JobStatus {
	return j.StatusValue
}

func (j JobStepTranscode) WithNewStatus(newStatus JobStatus, errorMsg *string) JobStep {
	j.StatusValue = newStatus
	j.LastError = *errorMsg
	return j
}

func (j JobStepTranscode) OutputId() *uuid.UUID {
	return j.ResultId
}

func (j JobStepTranscode) OutputData() interface{} {
	return nil
}

func (j JobStepTranscode) TimeTaken() float64 {
	return j.TimeTakenValue
}

func (j JobStepTranscode) ErrorMessage() string {
	return j.LastError
}

func (j JobStepTranscode) RunnerDesc() *JobRunnerDesc {
	return j.ContainerData
}

func (j JobStepTranscode) Store(redisClient *redis.Client) error {
	j.JobStepType = "transcode"
	dbKey := fmt.Sprintf("mediaflipper:jobsteptranscode:%s", j.JobStepId.String())
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

func (j JobStepTranscode) WithNewMediaFile(newMediaFile string) JobStep {
	j.MediaFile = newMediaFile
	j.StatusValue = JOB_PENDING
	return j
}

func LoadJobStepTranscode(fromId uuid.UUID, redisClient *redis.Client) (*JobStepTranscode, error) {
	dbKey := fmt.Sprintf("mediaflipper:jobsteptranscode:%s", fromId.String())
	content, getErr := redisClient.Get(dbKey).Result()

	if getErr != nil {
		log.Printf("Could not load key for jobstep %s: %s", fromId.String(), getErr)
		return nil, getErr
	}

	var rtn JobStepTranscode
	marshalErr := json.Unmarshal([]byte(content), &rtn)
	if marshalErr != nil {
		log.Printf("Could not understand data for jobstep %s: %s", fromId.String(), marshalErr)
		log.Printf("Offending data was %s", content)
		return nil, marshalErr
	}

	return &rtn, nil
}
