package models

import (
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
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
		ResultId:               safeGetUUID(mapData["analysisResult"]),
		LastError:              safeGetString(mapData["errorMessage"]),
		MediaFile:              safeGetString(mapData["mediaFile"]),
		KubernetesTemplateFile: safeGetString(mapData["templateFile"]),
		StartTime:              TimeFromOptionalString(mapData["startTime"]),
		EndTime:                TimeFromOptionalString(mapData["endTime"]),
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

func (j JobStepAnalysis) OutputId() *uuid.UUID {
	return &j.ResultId
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
