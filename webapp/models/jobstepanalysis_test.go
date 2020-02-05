package models

import (
	"github.com/google/uuid"
	"testing"
)

func TestJobStepAnalysis_WithNewStatus(t *testing.T) {
	st := JobStepAnalysis{
		JobStepId:              uuid.UUID{},
		JobContainerId:         uuid.UUID{},
		ContainerData:          nil,
		StatusValue:            JOB_PENDING,
		ResultId:               uuid.New(),
		MediaFile:              "",
		KubernetesTemplateFile: "",
	}

	updated := st.WithNewStatus(JOB_COMPLETED, nil)
	if updated.Status() != JOB_COMPLETED {
		t.Errorf("Job status update did not work, expected %d got %d", JOB_COMPLETED, st.StatusValue)
	}
}
