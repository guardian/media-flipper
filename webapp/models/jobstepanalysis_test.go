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

//func TestLoadJobStepAnalysis(t *testing.T) {
//	s, err := miniredis.Run()
//	if err != nil {
//		panic(err)
//	}
//	defer s.Close()
//
//	testClient := redis.NewClient(&redis.Options{
//		Addr: s.Addr(),
//	})
//
//	jobStepId := uuid.New()
//	step := JobStepAnalysis{
//		JobStepId:              jobStepId,
//		JobContainerId:         uuid.New(),
//		ContainerData:          nil,
//		StatusValue:            JOB_STARTED,
//		ResultId:               uuid.New(),
//		MediaFile:              "path/to/something",
//		KubernetesTemplateFile: "mytemplate.yaml",
//	}
//
//	content, marshalErr := json.Marshal(step)
//	if marshalErr != nil {
//		t.Error("Could not marshal test data")
//		t.FailNow()
//	}
//
//	setErr := s.Set(fmt.Sprintf("mediaflipper:JobStepAnalysis:%s", jobStepId.String()), string(content))
//	if setErr != nil {
//		t.Error("Could not store test data")
//		t.FailNow()
//	}
//
//	result, getErr := LoadJobStepAnalysis(jobStepId, testClient)
//	if getErr != nil {
//		t.Error("Load returned unexpected error: ", getErr)
//	} else {
//		if *result != step {
//			t.Errorf("Loaded data did not match test data.")
//		}
//	}
//}
