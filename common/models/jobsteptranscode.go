package models

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"time"
)

type JobStepTranscode struct {
	JobStepType            string                `json:"stepType" mapstructure:"stepType"` //this field is vital so we can correctly unmarshal json data from the store
	JobStepId              uuid.UUID             `json:"id" mapstructure:"id"`
	JobContainerId         uuid.UUID             `json:"jobContainerId" mapstructure:"jobContainerId"`
	ContainerData          *JobRunnerDesc        `json:"containerData" mapstructure:"containerData"`
	StatusValue            JobStatus             `json:"jobStepStatus" mapstructure:"jobStepStatus"`
	LastError              string                `json:"errorMessage" mapstructure:"errorMessage"`
	MediaFile              string                `json:"mediaFile" mapstructure:"mediaFile"`
	ResultId               *uuid.UUID            `json:"transcodeResult" mapstructure:"transcodeResult"`
	TimeTakenValue         float64               `json:"timeTaken" mapstructure:"timeTaken"`
	KubernetesTemplateFile string                `json:"templateFile" mapstructure:"templateFile"`
	StartTime              *time.Time            `json:"startTime" mapstructure:"startTime"`
	EndTime                *time.Time            `json:"endTime" mapstructure:"startTime"`
	TranscodeSettings      TranscodeTypeSettings `json:"transcodeSettings" mapstructure:"transcodeSettings"`
	ItemType               helpers.BulkItemType  `json:"itemType"`
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
	if errorMsg != nil {
		j.LastError = *errorMsg
	}
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

func (j JobStepTranscode) WithNewMediaFile(newMediaFile string) JobStep {
	j.MediaFile = newMediaFile
	j.StatusValue = JOB_PENDING
	return j
}

func JobStepTranscodeFromMap(mapData map[string]interface{}) (*JobStepTranscode, error) {
	var rtn JobStepTranscode
	transcodeSettingsRaw := mapData["transcodeSettings"]
	delete(mapData, "transcodeSettings")
	err := CustomisedMapStructureDecode(mapData, &rtn)

	//log.Printf("DEBUG: raw transcode settings are %s", spew.Sdump(transcodeSettingsRaw))
	if err != nil {
		return nil, err
	}
	var transcodeSettingsAV JobSettings
	avErr := CustomisedMapStructureDecode(transcodeSettingsRaw, &transcodeSettingsAV)
	//log.Printf("DEBUG: attempt at decoding as AV: %s %s", spew.Sdump(transcodeSettingsAV), avErr)
	if avErr == nil && transcodeSettingsAV.isValid() {
		rtn.TranscodeSettings = transcodeSettingsAV
		return &rtn, nil
	}

	var transcodeSettingsImg TranscodeImageSettings
	imErr := CustomisedMapStructureDecode(transcodeSettingsRaw, &transcodeSettingsImg)
	//log.Printf("DEBUG: attempt at decoding as image: %s %s", spew.Sdump(transcodeSettingsImg), imErr)
	//log.Printf("settings valid? %t", transcodeSettingsImg.isValid())
	if imErr == nil && transcodeSettingsImg.isValid() {
		rtn.TranscodeSettings = transcodeSettingsImg
		return &rtn, nil
	}

	log.Printf("WARNING: transcode step %s from job %s has unrecognised settings", rtn.JobStepId, rtn.JobContainerId)
	return &rtn, nil
}
