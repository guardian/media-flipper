package main

import (
	"errors"
	"strconv"
)

type FormatAnalysis struct {
	StreamCount    int16   `json:"nb_streams"`
	ProgCount      int16   `json:"nb_programs"`
	FormatName     string  `json:"format_name"`
	FormatLongName string  `json:"format_long_name"`
	StartTimeCode  float64 `json:"start_time"`
	Duration       float64 `json:"duration"`
	Size           int64   `json:"size"`
	BitRate        float64 `json:"bit_rate"`
	ProbeScore     int32   `json:"probe_score"`
}

func safeParseFloat(from map[string]interface{}, key string, defaultValue float64) (float64, error) {
	stringVal, haveString := from[key].(string)
	if haveString {
		return strconv.ParseFloat(stringVal, 64)
	} else {
		return defaultValue, errors.New("value did not exist")
	}
}

func safeParseInt(from map[string]interface{}, key string, defaultValue int64) (int64, error) {
	stringVal, haveString := from[key].(string)
	if haveString {
		return strconv.ParseInt(stringVal, 10, 64)
	} else {
		return defaultValue, errors.New("value did not exist")
	}
}

/**
we need to perform this conversion manually because some of the json fields provided
come through as strings but we need them as numbers
*/
func FormatAnalysisFromMap(from map[string]interface{}) FormatAnalysis {
	startConverted, _ := safeParseFloat(from, "start_time", 0)
	durConverted, _ := safeParseFloat(from, "duration", 0)
	sizeConverted, _ := safeParseInt(from, "size", 0)
	brConverted, _ := safeParseFloat(from, "bit_rate", 0)

	return FormatAnalysis{
		StreamCount:    int16(from["nb_streams"].(float64)),
		ProgCount:      int16(from["nb_programs"].(float64)),
		FormatName:     from["format_name"].(string),
		FormatLongName: from["format_long_name"].(string),
		StartTimeCode:  startConverted,
		Duration:       durConverted,
		Size:           sizeConverted,
		BitRate:        brConverted,
		ProbeScore:     int32(from["probe_score"].(float64)),
	}
}

type AnalysisResult struct {
	Success      bool           `json:"successful"`
	Format       FormatAnalysis `json:"format"`
	ErrorMessage *string        `json:"errorMessage"`
}
