package jobrunner

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
)

type JobRunnerRequest struct {
	RequestId       uuid.UUID         `json:"requestId"`
	PredefinedType  string            `json:"predefinedType"`
	TemplateFile    string            `json:"templateFile"`
	Image           string            `json:"image"`
	ImagePullPolicy string            `json:"imagePullPolicy"`
	Command         []string          `json:"command"`
	Env             map[string]string `json:"env"`
	ForJob          models.JobEntry   `json:"forJob"`
}

type JobRunnerResult struct {
	success bool
}
