package main

import "strconv"

type FormatAnalysis struct {
	StreamCount    int16   `mapstructure:"nb_streams"`
	ProgCount      int16   `mapstructure:"nb_programs"`
	FormatName     string  `mapstructure:"format_name"`
	FormatLongName string  `mapstructure:"format_long_name"`
	StartTimeCode  float64 `mapstructure:"start_time"`
	Duration       float64 `mapstructure:"duration"`
	Size           int64   `mapstructure:"size"`
	BitRate        float64 `mapstructure:"bit_rate"`
	ProbeScore     int32   `mapstructure:"probe_score"`
}

/**
we need to perform this conversion manually because some of the json fields provided
come through as strings but we need them as numbers
*/
func FormatAnalysisFromMap(from map[string]interface{}) FormatAnalysis {
	startConverted, _ := strconv.ParseFloat(from["start_time"].(string), 64)
	durConverted, _ := strconv.ParseFloat(from["duration"].(string), 64)
	sizeConverted, _ := strconv.ParseInt(from["size"].(string), 10, 64)
	brConverted, _ := strconv.ParseFloat(from["bit_rate"].(string), 64)

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
	Success bool           `json:"successful"`
	Format  FormatAnalysis `json:"format"`
}
