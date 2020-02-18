package models

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"time"
)

type JobStepAnalysis struct {
	JobStepType            string               `json:"stepType" mapstructure:"stepType"`
	JobStepId              uuid.UUID            `json:"id" mapstructure:"id"`
	JobContainerId         uuid.UUID            `json:"jobContainerId" mapstructure:"jobContainerId"`
	ContainerData          *JobRunnerDesc       `json:"containerData" mapstructure:"containerData"`
	StatusValue            JobStatus            `json:"jobStepStatus" mapstructure:"jobStepStatus"`
	ResultId               uuid.UUID            `json:"analysisResult" mapstructure:"analysisResult"`
	LastError              string               `json:"errorMessage" mapstructure:"errorMessage"`
	MediaFile              string               `json:"mediaFile" mapstructure:"mediaFile"`
	KubernetesTemplateFile string               `json:"templateFile" mapstructure:"templateFile"`
	StartTime              *time.Time           `json:"startTime" mapstructure:"startTime"`
	EndTime                *time.Time           `json:"endTime" mapstructure:"startTime"`
	ItemType               helpers.BulkItemType `json:"itemType"`
}

func JobStepAnalysisFromMap(mapData map[string]interface{}) (*JobStepAnalysis, error) {
	var rtn JobStepAnalysis
	err := CustomisedMapStructureDecode(mapData, &rtn)
	return &rtn, err
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
