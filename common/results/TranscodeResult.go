package results

type TranscodeResult struct {
	OutFile      string  `json:"outFile"`
	TimeTaken    float64 `json:"timeTaken"`
	ErrorMessage string  `json:"errorMessage"`
}
