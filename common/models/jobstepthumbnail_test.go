package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"testing"
)

func TestJobStepThumbnailFromMap(t *testing.T) {
	rawData := map[string]interface{}{
		"stepType":              "thumbnail",
		"id":                    "C77A8418-07C1-4545-805D-1A2EA3C7A851",
		"jobContainerId":        "EA141D8C-B6A5-434C-A2D3-A9F8E20DE598",
		"containerData":         nil,
		"jobStepStatus":         JOB_STARTED,
		"errorMessage":          "",
		"mediaFile":             "/path/to/media",
		"thumbnailFrameSeconds": 0,
		"thumbnailResult":       nil,
		"timeTaken":             0.2,
		"templateFile":          "sometemplate.yaml",
		"transcodeSettings": map[string]interface{}{
			"name":       "sample image settings",
			"settingsId": "9937EB0E-E075-4556-A916-8FC13D72A1F5",
			"scale_x":    1200,
			"scale_y":    1200,
		},
		"itemType": helpers.ITEM_TYPE_IMAGE,
	}

	result, err := JobStepThumbnailFromMap(rawData)
	if err != nil {
		t.Error("JobStepThumbnailFromMap errored unexpectedly: ", err)
	} else {
		if result.JobStepType != "thumbnail" {
			t.Errorf("JobStepType was incorrect, expected 'thumbnail' got '%s'", result.JobStepType)
		}
		if result.JobStepId != uuid.MustParse("C77A8418-07C1-4545-805D-1A2EA3C7A851") {
			t.Errorf("JobStepId was incorrect, expected C77A8418-07C1-4545-805D-1A2EA3C7A851 got %s", result.JobStepId)
		}
		if result.JobContainerId != uuid.MustParse("EA141D8C-B6A5-434C-A2D3-A9F8E20DE598") {
			t.Errorf("JobContainerId was incorrect, expected EA141D8C-B6A5-434C-A2D3-A9F8E20DE598 got %s", result.JobContainerId)
		}
		if result.ContainerData != nil {
			t.Errorf("ContainerData was incorrect, expected nil got %s", spew.Sdump(result.ContainerData))
		}
		if result.StatusValue != JOB_STARTED {
			t.Errorf("StatusValue was incorrect, expected %d got %d", JOB_STARTED, result.StatusValue)
		}
		if result.LastError != "" {
			t.Errorf("LastError was incorrect, expected '' got '%s'", result.LastError)
		}
		if result.MediaFile != "/path/to/media" {
			t.Errorf("MediaFile was incorrect, expected '/path/to/media' got '%s'", result.MediaFile)
		}
		if result.ThumbnailFrameSeconds != 0 {
			t.Errorf("ThumbnailFrameSeconds was incorrect, expected 0 got %f", result.ThumbnailFrameSeconds)
		}
		if result.ResultId != nil {
			t.Errorf("ResultId was incorrect, expected nil got %s", spew.Sdump(result.ResultId))
		}
		if result.TimeTakenValue != 0.2 {
			t.Errorf("TimeTakenValue was incorrect, expected 0.2 got %f", result.TimeTakenValue)
		}
		if result.KubernetesTemplateFile != "sometemplate.yaml" {
			t.Errorf("KubernetesTemplateFile was incorrect, expected 'sometemplate.yaml' got %s", result.KubernetesTemplateFile)
		}
		expectedSettings := TranscodeImageSettings{
			SettingsId: uuid.MustParse("9937EB0E-E075-4556-A916-8FC13D72A1F5"),
			Name:       "sample image settings",
			ScaleX:     1200,
			ScaleY:     1200,
		}
		if result.TranscodeSettings != expectedSettings {
			t.Errorf("TranscodeSettings was incorrect, expected %s got %s", spew.Sdump(expectedSettings), spew.Sdump(result.TranscodeSettings))
		}
		if result.ItemType != helpers.ITEM_TYPE_IMAGE {
			t.Errorf("ItemType was incorrect, expected %s got %s", helpers.ITEM_TYPE_IMAGE, result.ItemType)
		}
	}
}
