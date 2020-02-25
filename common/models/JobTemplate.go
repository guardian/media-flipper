package models

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

type JobStepTemplateDefinition struct {
	Id                     uuid.UUID `yaml:"Id"`
	PredeterminedType      string    `yaml:"PredeterminedType"`
	KubernetesTemplateFile string    `yaml:"KubernetesTemplateFile"`
	InProgressLabel        string    `yaml:"InProgressLabel"`
	TranscodeSettingsId    string    `yaml:"TranscodeSettingsId"`
	ThumbnailFrameSeconds  float64   `yaml:"ThumbnailFrameSeconds"`
}

type JobTemplateDefinition struct {
	Id          uuid.UUID                   `yaml:"Id"`
	JobTypeName string                      `yaml:"Name"`
	Steps       []JobStepTemplateDefinition `yaml:"Steps"`
	OutputPath  string                      `yaml:"OutputPath"`
}

type TemplateManagerIF interface {
	NewJobContainer(templateId uuid.UUID, itemType helpers.BulkItemType) (*JobContainer, error)
	ListTemplates() []JobTemplateDefinition
	GetJob(jobId uuid.UUID) (JobTemplateDefinition, bool)
}

type JobTemplateManager struct {
	loadedTemplates      map[uuid.UUID]JobTemplateDefinition
	transcodeSettingsMgr *TranscodeSettingsManager
}

/**
build a new JobTemplateManager
*/
func NewJobTemplateManager(fromFilePath string, transcodeSettingsMgr *TranscodeSettingsManager) (*JobTemplateManager, error) {
	content, readErr := ioutil.ReadFile(fromFilePath)
	if readErr != nil {
		log.Printf("Could not read job template data from %s: %s", fromFilePath, readErr)
		return nil, readErr
	}

	var loadedContent []JobTemplateDefinition
	marshalErr := yaml.Unmarshal(content, &loadedContent)
	if marshalErr != nil {
		log.Printf("Could not understand data from %s: %s", fromFilePath, marshalErr)
		return nil, marshalErr
	}

	loadedTemplates := make(map[uuid.UUID]JobTemplateDefinition, len(loadedContent))

	for _, templateDef := range loadedContent {
		loadedTemplates[templateDef.Id] = templateDef
	}
	mgr := JobTemplateManager{
		loadedTemplates:      loadedTemplates,
		transcodeSettingsMgr: transcodeSettingsMgr,
	}
	return &mgr, nil
}

func (mgr JobTemplateManager) getTranscodeSettings(transcodeSettingsId string) (TranscodeTypeSettings, error) {
	var s TranscodeTypeSettings
	//spew.Dump(stepTemplate)
	if transcodeSettingsId != "" {
		id, uuidErr := uuid.Parse(transcodeSettingsId)
		if uuidErr != nil {
			errMsg := fmt.Sprintf("template step had an invalid transcode settings id, %s", transcodeSettingsId)
			return nil, errors.New(errMsg)
		} else {
			s = mgr.transcodeSettingsMgr.GetSetting(id)
			if s == nil {
				errMsg := fmt.Sprintf("template step has an invalid transcode settings id %s, nothing found that matches it", transcodeSettingsId)
				return nil, errors.New(errMsg)
			} else {
				return s, nil
			}
		}
	} else {
		return nil, errors.New("template step was missing transcode settings id!")
	}
}

func (mgr JobTemplateManager) NewJobContainer(templateId uuid.UUID, itemType helpers.BulkItemType) (*JobContainer, error) {
	tplEntry, tplExists := mgr.loadedTemplates[templateId]
	if !tplExists {
		return nil, errors.New(fmt.Sprintf("Request for non-existent template with id %s", templateId))
	}

	newContainerId := uuid.New()
	steps := make([]JobStep, len(tplEntry.Steps))

	for idx, stepTemplate := range tplEntry.Steps {
		switch stepTemplate.PredeterminedType {
		case "analysis":
			newStep := JobStepAnalysis{
				JobStepType:            "analysis",
				JobStepId:              uuid.New(),
				JobContainerId:         newContainerId,
				ContainerData:          nil,
				StatusValue:            JOB_PENDING,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
				ItemType:               itemType,
			}
			steps[idx] = newStep
		case "thumbnail":
			s, settingsErr := mgr.getTranscodeSettings(stepTemplate.TranscodeSettingsId)
			if settingsErr != nil {
				log.Printf("WARNING: Could not get transocde settings for %s: %s", spew.Sdump(stepTemplate), settingsErr)
				s = nil
			}

			newStep := JobStepThumbnail{
				JobStepType:            "thumbnail",
				JobStepId:              uuid.New(),
				JobContainerId:         newContainerId,
				ContainerData:          nil,
				StatusValue:            JOB_PENDING,
				ThumbnailFrameSeconds:  stepTemplate.ThumbnailFrameSeconds,
				ResultId:               nil,
				TimeTakenValue:         0,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
				TranscodeSettings:      s,
				ItemType:               itemType,
			}
			steps[idx] = newStep
		case "transcode":
			s, settingsErr := mgr.getTranscodeSettings(stepTemplate.TranscodeSettingsId)
			if settingsErr != nil {
				log.Printf("WARNING: Could not get transocde settings for %s: %s", spew.Sdump(stepTemplate), settingsErr)
				s = nil
			}

			newStep := JobStepTranscode{
				JobStepType:            "transcode",
				JobStepId:              uuid.New(),
				JobContainerId:         newContainerId,
				ContainerData:          nil,
				StatusValue:            JOB_PENDING,
				ResultId:               nil,
				TimeTakenValue:         0,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
				TranscodeSettings:      s,
				ItemType:               itemType,
			}
			steps[idx] = newStep
		case "custom":
			newStep := JobStepCustom{
				JobStepType:            "custom",
				JobStepId:              uuid.New(),
				JobContainerId:         newContainerId,
				StatusValue:            JOB_PENDING,
				LastError:              "",
				StartTime:              nil,
				EndTime:                nil,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
				ItemType:               itemType,
			}
			steps[idx] = newStep
		default:
			log.Printf("ERROR: Unrecognised predetermined type: %s", stepTemplate.PredeterminedType)
		}
	}

	startTime := time.Now()
	return &JobContainer{
		Id:             newContainerId,
		JobTemplateId:  templateId,
		Steps:          steps,
		CompletedSteps: 0,
		Status:         JOB_PENDING,
		StartTime:      &startTime,
		OutputPath:     tplEntry.OutputPath,
	}, nil
}

func (mgr JobTemplateManager) ListTemplates() []JobTemplateDefinition {
	rtn := make([]JobTemplateDefinition, len(mgr.loadedTemplates))
	i := 0
	for _, templateDef := range mgr.loadedTemplates {
		rtn[i] = templateDef
		i += 1
	}
	return rtn
}

func (mgr JobTemplateManager) GetJob(jobId uuid.UUID) (JobTemplateDefinition, bool) {
	template, exists := mgr.loadedTemplates[jobId]
	return template, exists
}
