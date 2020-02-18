package main

import (
	"strings"
	"testing"
)

func TestParseSettings(t *testing.T) {
	imageSettingsString := `{"name":"sampleimage","description":"sample image settings","settingsId":"2C882CAA-386D-4963-91E1-FAA50DC84AED","scale_x":1200,"scale_y":1200}`

	result, imErr := ParseSettings(imageSettingsString)
	if imErr != nil {
		t.Error("ParseSettings unexpectedly failed: ", imErr)
	} else {
		stringOut := result.MarshalToString()
		if !strings.Contains(stringOut, "-resize 1200x1200") {
			t.Errorf("string output did not contain expected content. Got '%s'", stringOut)
		}
	}

	vidSettingsString := `{"name":"samplevideo","description":"sample video settings","settingsId":"2C882CAA-386D-4963-91E1-FAA50DC84AED","wrapper":{"format":"mp4"}}`

	vResult, vErr := ParseSettings(vidSettingsString)
	if vErr != nil {
		t.Error("ParseSettings unexpectedly failed: ", vErr)
	} else {
		stringOut := vResult.MarshalToString()
		if !strings.Contains(stringOut, "-f mp4") {
			t.Errorf("string output did not contain expected content. Got '%s'", stringOut)
		}
	}
}
