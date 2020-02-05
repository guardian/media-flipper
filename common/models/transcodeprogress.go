package models

import (
	"github.com/davecgh/go-spew/spew"
	"regexp"
	"strconv"
	"strings"
)

type TranscodeProgress struct {
	FramesProcessed int64   `json:"framesProcessed"`
	FramesPerSecond int32   `json:"fps"`
	QFactor         float32 `json:"qfactor"`
	SizeEncoded     int64   `json:"sizeEncoded"`
	TimeEncoded     float64 `json:"timeEncoded"`
	Bitrate         float64 `json:"bitrate"`
	SpeedFactor     float32 `json:"speedFactor"`
}

type NoMatchError struct {
}

func (e *NoMatchError) Error() string {
	return "string did not match expected format"
}

func getMultiplierFrom(mulString string) int64 {
	lowered := strings.ToLower(mulString)
	if strings.HasPrefix(lowered, "k") {
		return 1024
	} else if strings.HasPrefix(lowered, "m") {
		return 1048576
	} else if strings.HasPrefix(lowered, "g") {
		return 1073741824
	} else if strings.HasPrefix(lowered, "t") {
		return 1099511627776
	} else {
		return 1
	}
}

var progressParser = regexp.MustCompile(`^frame=\s*(?P<frameNo>\d+)\s*fps=\s*(?P<fps>\d+)\s*q=\s*(?P<qfactor>[\d\.]+)\s*.size=\s*(?P<encodedSizeBytes>\d+)(?P<encodedSizeMultiplier>\w+)\s*time=(?P<encTimeHrs>\d{2}):(?P<encTimeMin>\d{2}):(?P<encTimeSec>[\d\.]+)\s*bitrate=\s*(?P<bitrateBytes>[\d\.]+)(?P<bitrateMultiplier>[\w/]+)\s*speed=\s*(?P<speedFactor>[\d\.]+)`)

/**
try to parse the given string as an output and build a TranscodeProgress struct
*/
func ParseTranscodeProgress(outputString string) (*TranscodeProgress, error) {
	match := progressParser.FindStringSubmatch(outputString)
	if match == nil {
		return nil, &NoMatchError{}
	}

	namedCaptures := make(map[string]string, progressParser.NumSubexp())
	for index, name := range progressParser.SubexpNames() {
		if index != 0 && name != "" {
			namedCaptures[name] = match[index]
		}
	}
	spew.Dump(namedCaptures)
	framesProcessed, _ := strconv.ParseInt(namedCaptures["frameNo"], 10, 64)
	fps, _ := strconv.ParseInt(namedCaptures["fps"], 10, 32)
	qFac, _ := strconv.ParseFloat(namedCaptures["qfactor"], 32)
	encSizeBytes, _ := strconv.ParseInt(namedCaptures["encodedSizeBytes"], 10, 64)
	encSizeMul := getMultiplierFrom(namedCaptures["encodedSizeMultiplier"])
	encTimeHrs, _ := strconv.ParseInt(namedCaptures["encTimeHrs"], 10, 8)
	encTimeMin, _ := strconv.ParseInt(namedCaptures["encTimeMin"], 10, 8)
	encTimeSec, _ := strconv.ParseFloat(namedCaptures["encTimeSec"], 64)
	bitrateBytes, _ := strconv.ParseFloat(namedCaptures["bitrateBytes"], 64)
	bitrateMul := getMultiplierFrom(namedCaptures["bitrateMultiplier"])
	speedFactor, _ := strconv.ParseFloat(namedCaptures["speedFactor"], 32)

	result := TranscodeProgress{
		FramesProcessed: framesProcessed,
		FramesPerSecond: int32(fps),
		QFactor:         float32(qFac),
		SizeEncoded:     encSizeBytes * encSizeMul,
		TimeEncoded:     (float64(encTimeHrs) * 3600.0) + (float64(encTimeMin) * 60.0) + encTimeSec,
		Bitrate:         bitrateBytes * float64(bitrateMul),
		SpeedFactor:     float32(speedFactor),
	}
	spew.Dump(result)
	return &result, nil
}
