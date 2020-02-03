package models

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

type JobStepTemplateDefinition struct {
	Id                     uuid.UUID `yaml:"Id"`
	PredeterminedType      string    `yaml:"PredeterminedType"`
	KubernetesTemplateFile string    `yaml:"KubernetesTemplateFile"`
	TranscodeSettingsId    string    `yaml:"TranscodeSettingsId"`
}

type JobTemplateDefinition struct {
	Id          uuid.UUID                   `yaml:"Id"`
	JobTypeName string                      `yaml:"Name"`
	Steps       []JobStepTemplateDefinition `yaml:"Steps"`
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

func (mgr JobTemplateManager) NewJobContainer(settingsId uuid.UUID, templateId uuid.UUID) (*JobContainer, error) {
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
			}
			steps[idx] = newStep
			break
		case "thumbnail":
			newStep := JobStepThumbnail{
				JobStepType:            "thumbnail",
				JobStepId:              uuid.New(),
				JobContainerId:         newContainerId,
				ContainerData:          nil,
				StatusValue:            JOB_PENDING,
				ResultId:               nil,
				TimeTakenValue:         0,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
			}
			steps[idx] = newStep
			break
		case "transcode":
			var s *models.JobSettings
			spew.Dump(stepTemplate)
			if stepTemplate.TranscodeSettingsId != "" {
				uuid, uuidErr := uuid.Parse(stepTemplate.TranscodeSettingsId)
				if uuidErr != nil {
					log.Printf("template step had an invalid transcode settings id, %s", stepTemplate.TranscodeSettingsId)
					s = nil
				} else {
					s = mgr.transcodeSettingsMgr.GetSetting(uuid)
					if s == nil {
						log.Printf("template step has an invalid transcode settings id %s, nothing found that matches it", stepTemplate.TranscodeSettingsId)
					}
				}
			} else {
				log.Printf("template step was missing transcode settings id!")
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
			}
			steps[idx] = newStep
			break
		case "custom":
			log.Printf("custom type not implemented yet")
			break
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
