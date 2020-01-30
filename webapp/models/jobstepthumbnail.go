package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"log"
	"time"
)

type ThumbnailResult struct {
	OutPath      *string `json:"outPath",struct:"outPath"`
	ErrorMessage *string `json:"errorMessage",struct:"errorMessage"`
	TimeTaken    float64 `json:"timeTaken",struct:"timeTaken"`
}

type JobStepThumbnail struct {
	JobStepType            string           `json:"stepType"` //this field is vital so we can correctly unmarshal json data from the store
	JobStepId              uuid.UUID        `json:"id"`
	JobContainerId         uuid.UUID        `json:"jobContainerId"`
	ContainerData          *JobRunnerDesc   `json:"containerData"`
	StatusValue            JobStatus        `json:"jobStepStatus"`
	LastError              string           `json:"errorMessage",struct:"errorMessage"`
	MediaFile              string           `json:"mediaFile",struct:"mediaFile"`
	Result                 *ThumbnailResult `json:"thumbnailResult"`
	KubernetesTemplateFile string           `json:"templateFile"`
	StartTime              *time.Time       `json:"startTime",struct:"startTime"`
	EndTime                *time.Time       `json:"endTime",struct:"startTime"`
}

func JobStepThumbnailFromMap(mapData map[string]interface{}) (*JobStepThumbnail, error) {
	stepId, stepIdParseErr := uuid.Parse(mapData["id"].(string))
	if stepIdParseErr != nil {
		return nil, stepIdParseErr
	}
	contId, contIdParseErr := uuid.Parse(mapData["jobContainerId"].(string))
	if contIdParseErr != nil {
		return nil, contIdParseErr
	}

	var runnerDescPtr *JobRunnerDesc
	if mapData["containerData"] == nil {
		runnerDescPtr = nil
	} else {
		contDecodeErr := mapstructure.Decode(mapData["containerData"], runnerDescPtr)
		if contDecodeErr != nil {
			return nil, contDecodeErr
		}
	}

	var aResult ThumbnailResult
	resultDecodeErr := mapstructure.Decode(mapData["thumbnailResult"], &aResult)
	if resultDecodeErr != nil {
		return nil, resultDecodeErr
	}

	rtn := JobStepThumbnail{
		JobStepType:            "thumbnail",
		JobStepId:              stepId,
		JobContainerId:         contId,
		ContainerData:          runnerDescPtr,
		StatusValue:            JobStatus(mapData["jobStepStatus"].(float64)),
		Result:                 &aResult,
		MediaFile:              safeGetString(mapData["mediaFile"]),
		KubernetesTemplateFile: mapData["templateFile"].(string),
		StartTime:              timeFromOptionalString(mapData["startTime"]),
		EndTime:                timeFromOptionalString(mapData["endTime"]),
	}
	return &rtn, nil
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
	j.JobStepType = "thumbnail"
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
	if errMsg != nil {
		j.LastError = *errMsg
	}
	nowTime := time.Now()
	switch j.StatusValue {
	case JOB_STARTED:
		j.StartTime = &nowTime
		break
	case JOB_FAILED:
		fallthrough
	case JOB_COMPLETED:
		j.EndTime = &nowTime
		break
	default:
		break
	}
	return j
}

func (j JobStepThumbnail) WithNewMediaFile(newMediaFile string) JobStep {
	j.MediaFile = newMediaFile
	return j
}

func (j JobStepThumbnail) ContainerId() uuid.UUID {
	return j.JobContainerId
}