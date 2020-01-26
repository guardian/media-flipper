package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

type ThumbnailResult struct {
	OutPath      *string `json:"outPath"`
	ErrorMessage *string `json:"errorMessage"`
	TimeTaken    float64 `json:"timeTaken"`
}

type JobStepThumbnail struct {
	JobStepId              uuid.UUID        `json:"id"`
	JobContainerId         uuid.UUID        `json:"jobContainerId"`
	ContainerData          *JobRunnerDesc   `json:"containerData"`
	StatusValue            JobStatus        `json:"jobStepStatus"`
	Result                 *ThumbnailResult `json:"thumbnailResult"`
	KubernetesTemplateFile string           `json:"templateFile"`
}

func (j JobStepThumbnail) StepId() uuid.UUID {
	return j.JobStepId
}

func (j JobStepThumbnail) Status() JobStatus {
	return j.StatusValue
}

func (j JobStepThumbnail) OutputPath() string {
	if j.Result != nil {
		if j.Result.OutPath != nil {
			return ""
		} else {
			return *j.Result.OutPath
		}
	} else {
		return ""
	}
}

func (j JobStepThumbnail) OutputData() interface{} {
	return nil
}

func (j JobStepThumbnail) RunnerDesc() *JobRunnerDesc {
	return j.ContainerData
}

func (j JobStepThumbnail) TimeTaken() float64 {
	if j.Result != nil {
		return j.Result.TimeTaken
	} else {
		return -1
	}
}

func (j JobStepThumbnail) ErrorMessage() string {
	if j.Result != nil && j.Result.ErrorMessage != nil {
		return *j.Result.ErrorMessage
	} else {
		return ""
	}
}

func (j JobStepThumbnail) Store(redisClient *redis.Client) error {
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

func (j JobStepThumbnail) WithNewStatus(newStatus JobStatus, errMsg *string) JobStep {
	j.StatusValue = newStatus
	return j
}

func (j JobStepThumbnail) WithNewMediaFile(newMediaFile string) JobStep {
	return j
}

func (j JobStepThumbnail) ContainerId() uuid.UUID {
	return j.JobContainerId
}
