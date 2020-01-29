package models

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

type JobStepTemplateDefinition struct {
	Id                     uuid.UUID `yaml:"Id"`
	PredeterminedType      string    `yaml:"PredeterminedType"`
	KubernetesTemplateFile string    `yaml:"KubernetesTemplateFile"`
}

type JobTemplateDefinition struct {
	Id          uuid.UUID                   `yaml:"Id"`
	JobTypeName string                      `yaml:"Name"`
	Steps       []JobStepTemplateDefinition `yaml:"Steps"`
}

type JobTemplateManager struct {
	loadedTemplates map[uuid.UUID]JobTemplateDefinition
}

/**
build a new JobTemplateManager
*/
func NewJobTemplateManager(fromFilePath string) (*JobTemplateManager, error) {
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
		loadedTemplates: loadedTemplates,
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
				JobStepId:              stepTemplate.Id,
				JobContainerId:         newContainerId,
				ContainerData:          nil,
				StatusValue:            JOB_PENDING,
				Result:                 nil,
				MediaFile:              "",
				KubernetesTemplateFile: stepTemplate.KubernetesTemplateFile,
			}
			steps[idx] = newStep
			break
		case "transcode":
			log.Printf("transcode type not implemented yet")
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
