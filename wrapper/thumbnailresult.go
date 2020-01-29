package main

type ThumbnailResult struct {
	OutPath      *string `json:"outPath"`
	ErrorMessage *string `json:"errorMessage"`
	TimeTaken    float64 `json:"timeTaken"`
}
