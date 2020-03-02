package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"testing"
)

func TestJobStepTranscodeFromMapAV(t *testing.T) {
	//JobStepTranscodeFromMap should marshal all of the data from a struct
	fakeTranscodeSettings := map[string]interface{}{
		"settingsId":  "87230B0F-8E75-474D-B8D0-5C421C9D4E56",
		"name":        "test",
		"description": "test fake job settings",
		"wrapper": map[string]interface{}{
			"format": "mp4",
		},
	}
	mappedData := map[string]interface{}{
		"stepType":          "transcode",
		"id":                "088C9988-4FE3-4BC8-A0D3-55556AB0A922",
		"jobContainerId":    "E40A69A6-324E-48D9-AC18-6D9BA44E16B5",
		"containerData":     nil,
		"jobStepStatus":     1.0,
		"errorMessage":      "blahblah",
		"mediaFile":         "path/to/some/media",
		"transcodeResult":   "2EF99A08-7AF6-45F1-B34E-A60C01FBF5B2",
		"timeTaken":         4.567,
		"templateFile":      "sometemplate.yaml",
		"startTime":         "2020-02-03T04:05:06Z",
		"endTime":           "2020-02-03T05:06:07Z",
		"transcodeSettings": fakeTranscodeSettings,
	}

	result, err := JobStepTranscodeFromMap(mappedData)
	if err != nil {
		t.Error("JobStepTranscodeFromMap failed unexpectedly ", err)
	} else {
		if result.JobStepType != "transcode" {
			t.Errorf("Got wrong step type, expected 'transcode' got '%s'", result.JobStepType)
		}
		if result.JobStepId != uuid.MustParse("088C9988-4FE3-4BC8-A0D3-55556AB0A922") {
			t.Errorf("Got wrong step ID, expected 088C9988-4FE3-4BC8-A0D3-55556AB0A922 got %s", result.JobStepId.String())
		}
		if result.JobContainerId != uuid.MustParse("E40A69A6-324E-48D9-AC18-6D9BA44E16B5") {
			t.Errorf("Got wrong container ID, expected E40A69A6-324E-48D9-AC18-6D9BA44E16B5 got %s", result.JobContainerId.String())
		}
		if result.ContainerData != nil {
			t.Errorf("Expected containerData to be nil, got %s", spew.Sdump(result.ContainerData))
		}
		if result.StatusValue != JOB_STARTED {
			t.Errorf("Expected status to be %d, got %d", JOB_STARTED, result.StatusValue)
		}
		if result.LastError != "blahblah" {
			t.Errorf("Expected LastERror to be %s, got %s", "blahblah", result.LastError)
		}
		if result.MediaFile != "path/to/some/media" {
			t.Errorf("Expected MediaFile to be path/to/some/media, got %s", result.MediaFile)
		}
		if result.TranscodeSettings == nil {
			t.Errorf("Expected TranscodeSettings to be non-nil")
		} else {
			settings := result.TranscodeSettings.(JobSettings)
			if settings.Name != "test" {
				t.Errorf("Got wrong transcode settings name, expected 'test' got %s", settings.Name)
			}
			if settings.Description != "test fake job settings" {
				t.Errorf("Got wrong transcode settings description: %s", settings.Description)
			}
			if settings.SettingsId != uuid.MustParse("87230B0F-8E75-474D-B8D0-5C421C9D4E56") {
				t.Errorf("Got wrong transcode settings ID, expected 87230B0F-8E75-474D-B8D0-5C421C9D4E56 got %s", settings.SettingsId)
			}
		}
		//spew.Dump(result)
	}
}

func TestJobStepTranscodeFromMapImage(t *testing.T) {
	//JobStepTranscodeFromMap should marshal all of the data from a struct
	fakeTranscodeSettings := map[string]interface{}{
		"settingsId":  "87230B0F-8E75-474D-B8D0-5C421C9D4E56",
		"name":        "test",
		"description": "test fake job settings",
		"scale_x":     1200,
		"scale_y":     1200,
	}

	mappedData := map[string]interface{}{
		"stepType":          "transcode",
		"id":                "088C9988-4FE3-4BC8-A0D3-55556AB0A922",
		"jobContainerId":    "E40A69A6-324E-48D9-AC18-6D9BA44E16B5",
		"containerData":     nil,
		"jobStepStatus":     1.0,
		"errorMessage":      "blahblah",
		"mediaFile":         "path/to/some/media",
		"transcodeResult":   "2EF99A08-7AF6-45F1-B34E-A60C01FBF5B2",
		"timeTaken":         4.567,
		"templateFile":      "sometemplate.yaml",
		"startTime":         "2020-02-03T04:05:06Z",
		"endTime":           "2020-02-03T05:06:07Z",
		"transcodeSettings": fakeTranscodeSettings,
	}

	result, err := JobStepTranscodeFromMap(mappedData)
	if err != nil {
		t.Error("JobStepTranscodeFromMap failed unexpectedly ", err)
	} else {
		if result.JobStepType != "transcode" {
			t.Errorf("Got wrong step type, expected 'transcode' got '%s'", result.JobStepType)
		}
		if result.JobStepId != uuid.MustParse("088C9988-4FE3-4BC8-A0D3-55556AB0A922") {
			t.Errorf("Got wrong step ID, expected 088C9988-4FE3-4BC8-A0D3-55556AB0A922 got %s", result.JobStepId.String())
		}
		if result.JobContainerId != uuid.MustParse("E40A69A6-324E-48D9-AC18-6D9BA44E16B5") {
			t.Errorf("Got wrong container ID, expected E40A69A6-324E-48D9-AC18-6D9BA44E16B5 got %s", result.JobContainerId.String())
		}
		if result.ContainerData != nil {
			t.Errorf("Expected containerData to be nil, got %s", spew.Sdump(result.ContainerData))
		}
		if result.StatusValue != JOB_STARTED {
			t.Errorf("Expected status to be %d, got %d", JOB_STARTED, result.StatusValue)
		}
		if result.LastError != "blahblah" {
			t.Errorf("Expected LastERror to be %s, got %s", "blahblah", result.LastError)
		}
		if result.MediaFile != "path/to/some/media" {
			t.Errorf("Expected MediaFile to be path/to/some/media, got %s", result.MediaFile)
		}
		if result.TranscodeSettings == nil {
			t.Errorf("Expected TranscodeSettings to be non-nil")
		} else {
			settings := result.TranscodeSettings.(TranscodeImageSettings)
			if settings.Name != "test" {
				t.Errorf("Got wrong transcode settings name, expected 'test' got %s", settings.Name)
			}
			if settings.Description != "test fake job settings" {
				t.Errorf("Got wrong transcode settings description: %s", settings.Description)
			}
			if settings.SettingsId != uuid.MustParse("87230B0F-8E75-474D-B8D0-5C421C9D4E56") {
				t.Errorf("Got wrong transcode settings ID, expected 87230B0F-8E75-474D-B8D0-5C421C9D4E56 got %s", settings.SettingsId)
			}
		}
		//spew.Dump(result)
	}

}
