package models

/**
keep this in-sync with the corresponding file in wrapper
*/
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

type AnalysisResult struct {
	Success bool           `json:"successful"`
	Format  FormatAnalysis `json:"format"`
}
