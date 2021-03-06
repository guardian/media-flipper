package models

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"testing"
)

func TestNewJobTemplateManager(t *testing.T) {
	//NewJobTemplateManager should parse a YAML file and unmarshal it
	mgr, loadErr := NewJobTemplateManager("../../webapp/config/standardjobtemplate.yaml", nil)
	if loadErr != nil {
		t.Error("Load unexpectedly failed: ", loadErr)
		t.FailNow()
	}

	expectedUuid := uuid.MustParse("846F823E-C0D3-4AF0-AD51-0F9573379057")
	if len(mgr.loadedTemplates) != 3 {
		t.Errorf("Got %d templates, expected 3", len(mgr.loadedTemplates))
	}

	if mgr.loadedTemplates[expectedUuid].JobTypeName != "Standard thumbnail-and-transcode" {
		t.Errorf("Got unexpected jobTypeName: %s", mgr.loadedTemplates[expectedUuid].JobTypeName)
	}

	if len(mgr.loadedTemplates[expectedUuid].Steps) != 3 {
		t.Errorf("Got %d job steps, expected 3", len(mgr.loadedTemplates[expectedUuid].Steps))
	}

	//NewJobTemplateManager should return an error if it can't load the yaml
	_, shouldLoadErr := NewJobTemplateManager("fdsfsdjhsdfk", nil)
	if shouldLoadErr == nil {
		t.Error("Load should fail on an invalid filename")
	}
}

func TestNewJobContainer(t *testing.T) {
	settingsMgr, settingsLoadErr := NewTranscodeSettingsManager("../../webapp/config/settings")
	if settingsLoadErr != nil {
		t.Errorf("Could not load transcode settings: %s", settingsLoadErr)
		t.FailNow()
	}

	//NewJobContainer should create a JobContainer with a new UUID that links in a JobStep for each specified in the template
	mgr, loadErr := NewJobTemplateManager("../../webapp/config/standardjobtemplate.yaml", settingsMgr)
	if loadErr != nil {
		t.Error("Load unexpectedly failed: ", loadErr)
		t.FailNow()
	}

	expectedUuid := uuid.MustParse("846F823E-C0D3-4AF0-AD51-0F9573379057")
	result, err := mgr.NewJobContainer(expectedUuid, helpers.ITEM_TYPE_VIDEO)
	if err != nil {
		t.Error("NewJobContainer unexpectedly failed: ", err)
	} else {
		if result.Id == expectedUuid {
			t.Error("New container should not have the same uuid as the template")
		}
		if result.JobTemplateId != expectedUuid {
			t.Error("New container should store the template uuid, got ", result.JobTemplateId)
		}
		if len(result.Steps) != 3 {
			t.Error("Expected job to have 3 steps, got ", len(result.Steps))
		}

		analysisStep, isAnalysis := result.Steps[0].(JobStepAnalysis)
		if !isAnalysis {
			t.Error("Expected job step 1 to be analysis")
		}
		if analysisStep.KubernetesTemplateFile != "config/AnalysisJobTemplate.yaml" {
			t.Error("Got unexpected template file: ", analysisStep.KubernetesTemplateFile)
		}
		if analysisStep.JobContainerId != result.Id {
			t.Errorf("Step had incorrect container id, got %s expected %s", analysisStep.JobContainerId, result.Id)
		}
		if analysisStep.JobStepId == analysisStep.JobContainerId {
			t.Error("Job step id was the same as container ID")
		}

		thumbStep, isThumb := result.Steps[1].(JobStepThumbnail)
		if !isThumb {
			t.Error("Expected job step 1 to be analysis")
		}
		if thumbStep.KubernetesTemplateFile != "config/AnalysisJobTemplate.yaml" {
			t.Error("Got unexpected template file: ", analysisStep.KubernetesTemplateFile)
		}
		if thumbStep.JobContainerId != result.Id {
			t.Errorf("Step had incorrect container id, got %s expected %s", analysisStep.JobContainerId, result.Id)
		}
		if thumbStep.JobStepId == thumbStep.JobContainerId {
			t.Error("Job step id was the same as container ID")
		}

		tcStep, isTc := result.Steps[2].(JobStepTranscode)
		if !isTc {
			t.Error("Expected job step 2 to be transcode")
		} else {
			if tcStep.TranscodeSettings == nil {
				t.Errorf("Step 2 had no transcode settings, expecting some")
			} else {

			}
		}
	}
}

func TestJobContainer_SetMediaFile(t *testing.T) {

}
