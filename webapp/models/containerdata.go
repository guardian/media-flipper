package models

type JobRunnerDesc struct {
	JobUID         string `json:"jobUid"`
	Status         string `json:"status"`
	StartTime      string `json:"startTime"`
	CompletionTime string `json:"completionTime"`
	Name           string `json:"consoleName"`
}
