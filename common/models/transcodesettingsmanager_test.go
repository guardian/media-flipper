package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"testing"
)

func TestNewTranscodeSettingsManager(t *testing.T) {
	mgr, loadErr := NewTranscodeSettingsManager("../../webapp/config/settings")
	if loadErr != nil {
		t.Errorf("Couldn't initialise: %s", loadErr)
		t.FailNow()
	}

	if len(mgr.knownSettings) != 2 {
		t.Errorf("Wrong number of loaded settings, expected 2 got %d", len(mgr.knownSettings))
	}

	allSettings := mgr.ListSettings()
	if len(*allSettings) != len(mgr.knownSettings) {
		t.Errorf("Wrong number returned from ListSettings, expected %d got %d", len(mgr.knownSettings), len(*allSettings))
	}

	for _, s := range *allSettings {
		verifyData, isPresent := mgr.knownSettings[s.GetId()]
		if !isPresent {
			t.Errorf("setting with id %s was returned from ListSettings but is not present?!", s.GetId())
		} else {
			if verifyData != s {
				t.Errorf("mismatched setting returned from ListSettings, expected %s got %s", spew.Sprint(verifyData), spew.Sprint(s))
			}
		}
	}

	mp4SettingGeneric := mgr.GetSetting(uuid.MustParse("7FEC2963-6A1D-46A2-8DE1-62DF939F6755"))
	if mp4SettingGeneric == nil {
		t.Errorf("GetSetting returned nil but expected record")
	} else {
		mp4Setting := mp4SettingGeneric.(JobSettings)
		if mp4Setting.Video.Scale.ScaleY != -1 {
			t.Errorf("Wrong value for scaleY, expected -1 got %d", mp4Setting.Video.Scale.ScaleY)
		}
		if mp4Setting.Video.Scale.ScaleX != 1280 {
			t.Errorf("Wrong value for scaleX, expected 1280 got %d", mp4Setting.Video.Scale.ScaleX)
		}
	}
}
