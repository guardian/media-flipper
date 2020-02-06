package models

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func showTestAssertionError(testLine int, desc string, expected string, actual string, t *testing.T) {
	t.Errorf("%s did not match in line %d. Got %s, expected %s", desc, testLine, actual, expected)
}

func TestParseTranscodeProgress(t *testing.T) {
	sampleLines := []string{
		`frame= 1147 fps= 71 q=-1.0 Lsize=    8936kB time=00:00:45.88 bitrate=1595.3kbits/s speed=2.82x
video:8179kB audio:721kB subtitle:0kB other streams:0kB global headers:0kB muxing overhead: 0.398820% `,
		`frame=   53 fps=0.0 q=0.0 size=       0kB time=00:00:02.36 bitrate=   0.2kbits/s speed=4.69x
`,
		`frame= 3495 fps=109 q=-1.0 Lsize=   26121kB time=00:02:19.81 bitrate=1530.4kbits/s speed=4.37x
video:23824kB audio:2193kB subtitle:0kB other streams:0kB global headers:0kB muxing overhead: 0.402554%
`,
	}

	results := make([]TranscodeProgress, len(sampleLines))

	for i, line := range sampleLines {
		item, err := ParseTranscodeProgress(line)
		if err != nil {
			t.Errorf("could not parse string '%s': %s", line, err)
		} else {
			results[i] = *item
		}
	}

	if results[0].FramesProcessed != 1147 {
		showTestAssertionError(0, "FramesProcessed", "1147", strconv.FormatInt(results[0].FramesProcessed, 10), t)
	}
	if results[0].FramesPerSecond != 71 {
		showTestAssertionError(0, "FramesPerSecond", "71", strconv.FormatInt(int64(results[0].FramesPerSecond), 10), t)
	}
	if results[0].QFactor != -1 {
		showTestAssertionError(0, "QFactor", "-1", fmt.Sprintf("%f", results[0].QFactor), t)
	}
	if results[0].SizeEncoded != 9150464 {
		showTestAssertionError(0, "SizeEncoded", "9150464", fmt.Sprintf("%d", results[0].SizeEncoded), t)
	}
	if results[0].TimeEncoded != 45.88 {
		showTestAssertionError(0, "TimeEncoded", "45.88", fmt.Sprintf("%f", results[0].TimeEncoded), t)
	}
	if results[0].Bitrate != 1633587.2 {
		showTestAssertionError(0, "Bitrate", "1633587.2", fmt.Sprintf("%f", results[0].Bitrate), t)
	}
	if results[0].SpeedFactor != 2.82 {
		showTestAssertionError(0, "SpeedFactor", "2.82", fmt.Sprintf("%f", results[0].SpeedFactor), t)
	}

	if results[1].FramesProcessed != 53 {
		showTestAssertionError(1, "FramesProcessed", "53", strconv.FormatInt(results[0].FramesProcessed, 10), t)
	}
	if results[1].FramesPerSecond != 0 {
		showTestAssertionError(1, "FramesPerSecond", "0", strconv.FormatInt(int64(results[0].FramesPerSecond), 10), t)
	}
	if results[1].QFactor != 0 {
		showTestAssertionError(1, "QFactor", "0", fmt.Sprintf("%f", results[0].QFactor), t)
	}
	if results[1].SizeEncoded != 0 {
		showTestAssertionError(1, "SizeEncoded", "0", fmt.Sprintf("%d", results[0].SizeEncoded), t)
	}
	if results[1].TimeEncoded != 2.36 {
		showTestAssertionError(1, "TimeEncoded", "2.36", fmt.Sprintf("%f", results[0].TimeEncoded), t)
	}
	if results[1].Bitrate != 204.8 {
		showTestAssertionError(1, "Bitrate", "204.8", fmt.Sprintf("%f", results[0].Bitrate), t)
	}
	if results[1].SpeedFactor != 4.69 {
		showTestAssertionError(1, "SpeedFactor", "4.69", fmt.Sprintf("%f", results[0].SpeedFactor), t)
	}

	if results[2].FramesProcessed != 3495 {
		showTestAssertionError(2, "FramesProcessed", "3495", strconv.FormatInt(results[0].FramesProcessed, 10), t)
	}
	if results[2].FramesPerSecond != 109 {
		showTestAssertionError(2, "FramesPerSecond", "109", strconv.FormatInt(int64(results[0].FramesPerSecond), 10), t)
	}
	if results[2].QFactor != -1.0 {
		showTestAssertionError(2, "QFactor", "-1.0", fmt.Sprintf("%f", results[0].QFactor), t)
	}
	if results[2].SizeEncoded != 26747904 {
		showTestAssertionError(2, "SizeEncoded", "26747904", fmt.Sprintf("%d", results[0].SizeEncoded), t)
	}
	if results[2].TimeEncoded != 139.81 {
		showTestAssertionError(0, "TimeEncoded", "139.81", fmt.Sprintf("%f", results[0].TimeEncoded), t)
	}
	if results[2].Bitrate != 1567129.6 {
		showTestAssertionError(2, "Bitrate", "1567129.6", fmt.Sprintf("%f", results[0].Bitrate), t)
	}
	if results[2].SpeedFactor != 4.37 {
		showTestAssertionError(2, "SpeedFactor", "4.37", fmt.Sprintf("%f", results[0].SpeedFactor), t)
	}

	brokenLine := `fdsdsdfjhsdfsfsdfssfjhsdf`
	_, err := ParseTranscodeProgress(brokenLine)
	if err == nil {
		t.Errorf("Got no error from an invalid line")
	} else {
		_, isRightType := err.(*NoMatchError)
		if !isRightType {
			t.Errorf("Got an error %s for an invalid string, expected NoMatchError", reflect.TypeOf(err))
		}
	}
}

func TestGetMultiplierFrom(t *testing.T) {
	kmul := getMultiplierFrom("kbit/s")
	if kmul != 1024 {
		t.Errorf("Got %d for kb multiplier, expected 1024", kmul)
	}
	mmul := getMultiplierFrom("mbit/s")
	if mmul != 1048576 {
		t.Errorf("Got %d for mb multiplier, expected 1048576", mmul)
	}
	gmul := getMultiplierFrom("Gb")
	if gmul != 1073741824 {
		t.Errorf("Got %d for Gb multiplier, expected 1073741824", gmul)
	}
	tmul := getMultiplierFrom("Tb")
	if tmul != 1099511627776 {
		t.Errorf("Got %d for Tb multiplier, expected 1.099511628e12", tmul)
	}
	nomul := getMultiplierFrom("bytes")
	if nomul != 1 {
		t.Errorf("Got %d for null multiplier, expected 1", nomul)
	}
}
