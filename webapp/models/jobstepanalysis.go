package models

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"log"
	"time"
)

type JobStepAnalysis struct {
	JobStepType            string         `json:"stepType",struct:"stepType"`
	JobStepId              uuid.UUID      `json:"id",struct:"id"`
	JobContainerId         uuid.UUID      `json:"jobContainerId",struct:"jobContainerId"`
	ContainerData          *JobRunnerDesc `json:"containerData",struct:"containerData"`
	StatusValue            JobStatus      `json:"jobStepStatus",struct:"jobStepStatus"`
	ResultId               uuid.UUID      `json:"analysisResult",struct:"analysisResult"`
	LastError              string         `json:"errorMessage",struct:"errorMessage"`
	MediaFile              string         `json:"mediaFile",struct:"mediaFile"`
	KubernetesTemplateFile string         `json:"templateFile",struct:"templateFile"`
	StartTime              *time.Time     `json:"startTime",struct:"startTime"`
	EndTime                *time.Time     `json:"endTime",struct:"startTime"`
}

func safeGetString(from interface{}) string {
	if from == nil {
		return ""
	}
	stringContent, isString := from.(string)
	if !isString {
		log.Printf("WARNING: expected string, got %s", spew.Sdump(from))
		return ""
	}
	return stringContent
}

func getUUID(from interface{}) uuid.UUID {
	stringContent := safeGetString(from)
	if stringContent == "" {
		return uuid.UUID{}
	}
	parsed, parseErr := uuid.Parse(stringContent)
	if parseErr != nil {
		log.Printf("Could not decode UUID from '%s' (jobstepanalysis.go/getUUID)", parseErr)
		return uuid.UUID{}
	}
	return parsed
}

func JobStepAnalysisFromMap(mapData map[string]interface{}) (*JobStepAnalysis, error) {
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

	rtn := JobStepAnalysis{
		JobStepType:            "analysis",
		JobStepId:              stepId,
		JobContainerId:         contId,
		ContainerData:          runnerDescPtr,
		StatusValue:            JobStatus(mapData["jobStepStatus"].(float64)),
		ResultId:               getUUID(mapData["analysisResult"]),
		LastError:              safeGetString(mapData["errorMessage"]),
		MediaFile:              safeGetString(mapData["mediaFile"]),
		KubernetesTemplateFile: safeGetString(mapData["templateFile"]),
	}
	return &rtn, nil
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
	return j.ResultId
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
	j.JobStepType = "analysis"
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

func (j JobStepAnalysis) WithNewStatus(newStatus JobStatus, errMsg *string) JobStep {
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

func (j JobStepAnalysis) WithNewMediaFile(newMediaFile string) JobStep {
	j.MediaFile = newMediaFile
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
