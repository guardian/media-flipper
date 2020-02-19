package models

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"time"
)

type JobStepCustom struct {
	JobStepType            string               `json:"stepType" mapstructure:"stepType"` //this field is vital so we can correctly unmarshal json data from the store
	JobStepId              uuid.UUID            `json:"id" mapstructure:"id"`
	JobContainerId         uuid.UUID            `json:"jobContainerId" mapstructure:"jobContainerId"`
	StatusValue            JobStatus            `json:"jobStepStatus" mapstructure:"jobStepStatus"`
	LastError              string               `json:"errorMessage" mapstructure:"errorMessage"`
	StartTime              *time.Time           `json:"startTime" mapstructure:"startTime"`
	EndTime                *time.Time           `json:"endTime" mapstructure:"startTime"`
	MediaFile              string               `json:"mediaFile"`
	KubernetesTemplateFile string               `json:"templateFile" mapstructure:"templateFile"`
	ItemType               helpers.BulkItemType `json:"itemType"`
	CustomArguments        map[string]string    `json:"customArguments"`
}

func JobStepCustomFromMap(mapData map[string]interface{}) (*JobStepCustom, error) {
	var rtn JobStepCustom

	err := CustomisedMapStructureDecode(mapData, &rtn)

	return &rtn, err
}

func (j JobStepCustom) StepId() uuid.UUID {
	return j.JobStepId
}
func (j JobStepCustom) ContainerId() uuid.UUID {
	return j.JobContainerId
}
func (j JobStepCustom) Status() JobStatus {
	return j.StatusValue
}
func (j JobStepCustom) WithNewStatus(newStatus JobStatus, errorMsg *string) JobStep {
	j.StatusValue = newStatus
	if errorMsg != nil {
		j.LastError = *errorMsg
	}
	return j
}

func (j JobStepCustom) OutputId() *uuid.UUID {
	return nil //never any output from this
}
func (j JobStepCustom) OutputData() interface{} {
	return nil
}
func (j JobStepCustom) TimeTaken() float64 {
	if j.StartTime != nil || j.EndTime == nil {
		return 0.0
	}
	duration := j.EndTime.UnixNano() - j.StartTime.UnixNano()
	return float64(duration) / 1e9
}

func (j JobStepCustom) ErrorMessage() string {
	return j.LastError
}
func (j JobStepCustom) RunnerDesc() *JobRunnerDesc {
	return nil
}
func (j JobStepCustom) WithNewMediaFile(newMediaFile string) JobStep {
	j.MediaFile = newMediaFile
	return j
}
