package models

type ContainerStatus int

const (
	CONTAINER_ACTIVE ContainerStatus = iota
	CONTAINER_COMPLETED
	CONTAINER_FAILED
	CONTAINER_UNKNOWN_STATE
)

type JobRunnerDesc struct {
	JobUID         string          `json:"jobUid"`
	Status         ContainerStatus `json:"status"`
	StartTime      string          `json:"startTime"`
	CompletionTime string          `json:"completionTime"`
	Name           string          `json:"consoleName"`
}
