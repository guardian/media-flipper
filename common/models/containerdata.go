package models

type ContainerStatus int

const (
	CONTAINER_ACTIVE ContainerStatus = iota
	CONTAINER_COMPLETED
	CONTAINER_FAILED
	CONTAINER_UNKNOWN_STATE
)

/**
a more convenient representation of job metadata from the kubernetes server
this object is not stored, it's simply translated from the fuller data to make processin easier
*/
type JobRunnerDesc struct {
	JobUID         string          `json:"jobUid"`
	Status         ContainerStatus `json:"status"`
	StartTime      string          `json:"startTime"`
	CompletionTime string          `json:"completionTime"`
	Name           string          `json:"consoleName"`
}
