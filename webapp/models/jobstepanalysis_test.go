package models

import (
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"testing"
)

func TestJobStepAnalysis_Store(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr:               s.Addr(),
	})

	jobStepId := uuid.New()
	step := JobStepAnalysis{
		JobStepId:              jobStepId,
		JobContainerId:         uuid.New(),
		ContainerData:          nil,
		StatusValue:            JOB_STARTED,
		Result:                 AnalysisResult{},
		MediaFile:              "path/to/something",
		KubernetesTemplateFile: "mytemplate.yaml",
	}

	outputErr := step.Store(testClient)
	if outputErr != nil {
		t.Error("Store operation unexpectedly errored: ", outputErr)
	}

	content, getErr := s.Get(fmt.Sprintf("mediaflipper:JobStepAnalysis:%s", jobStepId.String()))

	if getErr != nil {
		t.Error("Could not retrieve stored key: ", getErr)
	} else {
		var storedStep JobStepAnalysis
		marshalErr := json.Unmarshal([]byte(content), &storedStep)
		if marshalErr != nil {
			t.Error("Could not unmarshal content: ", marshalErr)
		} else {
			if storedStep != step {
				t.Error("Stored data did not match test data")
			}
		}
	}
}

func TestJobStepAnalysis_WithNewStatus(t *testing.T) {
	st := JobStepAnalysis{
		JobStepId:              uuid.UUID{},
		JobContainerId:         uuid.UUID{},
		ContainerData:          nil,
		StatusValue:            JOB_PENDING,
		Result:                 AnalysisResult{},
		MediaFile:              "",
		KubernetesTemplateFile: "",
	}

	updated := st.WithNewStatus(JOB_COMPLETED)
	if updated.Status() != JOB_COMPLETED {
		t.Errorf("Job status update did not work, expected %d got %d", JOB_COMPLETED, st.StatusValue)
	}
}

func TestLoadJobStepAnalysis(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr:               s.Addr(),
	})

	jobStepId := uuid.New()
	step := JobStepAnalysis{
		JobStepId:              jobStepId,
		JobContainerId:         uuid.New(),
		ContainerData:          nil,
		StatusValue:            JOB_STARTED,
		Result:                 AnalysisResult{},
		MediaFile:              "path/to/something",
		KubernetesTemplateFile: "mytemplate.yaml",
	}

	content, marshalErr := json.Marshal(step)
	if marshalErr != nil {
		t.Error("Could not marshal test data")
		t.FailNow()
	}

	setErr := s.Set(fmt.Sprintf("mediaflipper:JobStepAnalysis:%s", jobStepId.String()), string(content))
	if setErr != nil {
		t.Error("Could not store test data")
		t.FailNow()
	}

	result, getErr := LoadJobStepAnalysis(jobStepId, testClient)
	if getErr != nil {
		t.Error("Load returned unexpected error: ", getErr)
	} else {
		if *result != step {
			t.Errorf("Loaded data did not match test data.")
		}
	}
}