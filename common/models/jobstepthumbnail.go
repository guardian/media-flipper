package models

import (
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"time"
)

type ThumbnailResult struct {
	OutPath      *string `json:"outPath" mapstructure:"outPath"`
	ErrorMessage *string `json:"errorMessage" mapstructure:"errorMessage"`
	TimeTaken    float64 `json:"timeTaken" mapstructure:"timeTaken"`
}

type JobStepThumbnail struct {
	JobStepType            string         `json:"stepType" mapstructure:"stepType"` //this field is vital so we can correctly unmarshal json data from the store
	JobStepId              uuid.UUID      `json:"id" mapstructure:"id"`
	JobContainerId         uuid.UUID      `json:"jobContainerId" mapstructure:"jobContainerId"`
	ContainerData          *JobRunnerDesc `json:"containerData" mapstructure:"containerData"`
	StatusValue            JobStatus      `json:"jobStepStatus" mapstructure:"jobStepStatus"`
	LastError              string         `json:"errorMessage" mapstructure:"errorMessage"`
	MediaFile              string         `json:"mediaFile" mapstructure:"mediaFile"`
	ThumbnailFrameSeconds  float64        `json:"thumbnailFrameSeconds" mapstructure:"thumbnailFrameSeconds"`
	ResultId               *uuid.UUID     `json:"thumbnailResult" mapstructure:"thumbnailResult"`
	TimeTakenValue         float64        `json:"timeTaken" mapstructure:"timeTaken"`
	KubernetesTemplateFile string         `json:"templateFile" mapstructure:"templateFile"`
	StartTime              *time.Time     `json:"startTime" mapstructure:"startTime"`
	EndTime                *time.Time     `json:"endTime" mapstructure:"endTime"`
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

	resultId := safeGetUUID(mapData["thumbnailResult"])
	rtn := JobStepThumbnail{
		JobStepType:            "thumbnail",
		JobStepId:              stepId,
		JobContainerId:         contId,
		ContainerData:          runnerDescPtr,
		StatusValue:            JobStatus(mapData["jobStepStatus"].(float64)),
		ResultId:               &resultId,
		ThumbnailFrameSeconds:  safeFloat(mapData["thumbnailFrameSeconds"], 0),
		TimeTakenValue:         safeFloat(mapData["timeTaken"], 0),
		MediaFile:              safeGetString(mapData["mediaFile"]),
		KubernetesTemplateFile: mapData["templateFile"].(string),
		StartTime:              TimeFromOptionalString(mapData["startTime"]),
		EndTime:                TimeFromOptionalString(mapData["endTime"]),
	}
	return &rtn, nil
}

func (j JobStepThumbnail) StepId() uuid.UUID {
	return j.JobStepId
}

func (j JobStepThumbnail) Status() JobStatus {
	return j.StatusValue
}

func (j JobStepThumbnail) OutputId() *uuid.UUID {
	return j.ResultId
}

func (j JobStepThumbnail) OutputData() interface{} {
	return nil
}

func (j JobStepThumbnail) RunnerDesc() *JobRunnerDesc {
	return j.ContainerData
}

func (j JobStepThumbnail) TimeTaken() float64 {
	return j.TimeTakenValue
}

func (j JobStepThumbnail) ErrorMessage() string {
	return j.LastError
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