package jobrunner

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
)

type JobRunnerRequest struct {
	requestId       uuid.UUID
	predefinedType  string
	templateFile    string
	image           string
	imagePullPolicy string
	command         []string
	env             map[string]string
	forJob          models.JobEntry
}

type JobRunnerResult struct {
	success bool
}
