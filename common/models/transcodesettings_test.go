package models

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	"testing"
)

/**
test that the transcodesettings object correctly unmarshals from sample data
*/
func TestTranscodeSettingsLoad(t *testing.T) {
	yamlData :=
		`- settingsid: "7FEC2963-6A1D-46A2-8DE1-62DF939F6755"
  name: mp4proxy
  description: Small MP4 file suitable for use as a video proxy
  wrapper:
    format: mp4
  audio:
    codec: libfaac
    bitrate: 128000
    channels: 2
    samplerate: 48000
  video:
    codec: h264
    bitrate: 1048576  #1mbit/s
    scale:
      scalex: 1280
      scaley: -1
      allowupscaling: false
`
	var settings []JobSettings
	err := yaml.Unmarshal([]byte(yamlData), &settings)
	if err != nil {
		t.Errorf("Could not unmarshal content from yaml: %s", err)
		t.FailNow()
	}

	if len(settings) != 1 {
		t.Errorf("Expected to get %d settings from data but got %d", 1, len(settings))
	} else {
		if settings[0].SettingsId != uuid.MustParse("7FEC2963-6A1D-46A2-8DE1-62DF939F6755") {
			t.Errorf("Got wrong uuid, got %s expected %s", settings[0].SettingsId.String(), "7FEC2963-6A1D-46A2-8DE1-62DF939F6755")
		}
		if settings[0].Name != "mp4proxy" {
			t.Errorf("Got wrong name, expected 'mp4proxy' got '%s'", settings[0].Name)
		}
		if settings[0].Description != "Small MP4 file suitable for use as a video proxy" {
			t.Errorf("Got wrong description '%s'", settings[0].Description)
		}
		if settings[0].Wrapper.Format != "mp4" {
			t.Errorf("Got wrong format name, expected 'mp4' got '%s'", settings[0].Wrapper.Format)
		}
		if settings[0].Audio.Codec != "libfaac" {
			t.Errorf("Got wrong audio codec, expected 'libfaac' got '%s'", settings[0].Audio.Codec)
		}
		if settings[0].Audio.Samplerate != 48000 {
			t.Errorf("Got wrong audio samplerate, expected 48000 got %d", settings[0].Audio.Samplerate)
		}
		if settings[0].Audio.Bitrate != 128000 {
			t.Errorf("Got wrong audio bitrate, expected 128000 got %d", settings[0].Audio.Bitrate)
		}
		if settings[0].Audio.Channels != 2 {
			t.Errorf("Got wrong audio channels, expected 2 got %d", settings[0].Audio.Channels)
		}
		if settings[0].Video.Codec != "h264" {
			t.Errorf("Got wrong video codec, expected 'h264' got %s", settings[0].Video.Codec)
		}
		if settings[0].Video.Bitrate != 1048576 {
			t.Errorf("Got wrong video bitrate, expected 1048576 got %d", settings[0].Video.Bitrate)
		}
		if settings[0].Video.Scale.AllowUpscaling {
			t.Errorf("Expected scale.allowUpscaling to be false, got true")
		}
		if settings[0].Video.Scale.ScaleY != -1 {
			t.Errorf("Expected scale.scaleY to be -1, got %d", settings[0].Video.Scale.ScaleY)
		}
		if settings[0].Video.Scale.ScaleX != 1280 {
			t.Errorf("Expected scale.scaleX to be 1280, got %d", settings[0].Video.Scale.ScaleX)
		}
	}
}