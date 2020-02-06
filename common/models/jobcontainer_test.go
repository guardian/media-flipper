package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"testing"
)

/**
InitialStep should return the first step in the list
*/
func TestJobContainer_InitialStep(t *testing.T) {
	containerId := uuid.New()
	steps := []JobStep{
		JobStepAnalysis{
			JobStepId:              uuid.UUID{},
			JobContainerId:         containerId,
			ContainerData:          nil,
			StatusValue:            0,
			ResultId:               uuid.New(),
			MediaFile:              "",
			KubernetesTemplateFile: "",
		},
		JobStepThumbnail{
			JobStepId:              uuid.UUID{},
			JobContainerId:         containerId,
			ContainerData:          nil,
			StatusValue:            0,
			ResultId:               nil,
			KubernetesTemplateFile: "",
		},
	}

	container := JobContainer{
		Id:             containerId,
		Steps:          steps,
		CompletedSteps: 0,
		Status:         JOB_PENDING,
		JobTemplateId:  uuid.UUID{},
	}

	result := container.InitialStep()
	if result != steps[0] {
		t.Errorf("Got %s for initial step, expected %s", spew.Sprint(result), spew.Sprint(steps[0]))
	}
	if container.Status != JOB_PENDING {
		t.Errorf("Container status changed to %d, expected %d", container.Status, JOB_PENDING)
	}
}

/**
container should auto-complete and return nil if there are no job steps
*/
func TestJobContainer_InitialStepEmpty(t *testing.T) {
	containerId := uuid.New()
	steps := []JobStep{}

	container := JobContainer{
		Id:             containerId,
		Steps:          steps,
		CompletedSteps: 0,
		Status:         JOB_PENDING,
		JobTemplateId:  uuid.UUID{},
	}

	result := container.InitialStep()
	if result != nil {
		t.Errorf("Got %s for initial step, expected nil", spew.Sprint(result))
	}

	if container.Status != JOB_COMPLETED {
		t.Errorf("Got %d for container status after test, expected %d", container.Status, JOB_COMPLETED)
	}

	if container.EndTime == nil {
		t.Errorf("Expected container completed time to be set")
	}
}

/**
CompleteStepAndMoveOn should update the completed steps counter and return the next step in the list.
It should return nil and set the status to JOB_COMPLETED when we reach the end of the list
*/
func TestJobContainer_CompleteStepAndMoveOn(t *testing.T) {
	containerId := uuid.New()
	steps := []JobStep{
		JobStepAnalysis{
			JobStepId:              uuid.UUID{},
			JobContainerId:         containerId,
			ContainerData:          nil,
			StatusValue:            0,
			ResultId:               uuid.New(),
			MediaFile:              "",
			KubernetesTemplateFile: "",
		},
		JobStepThumbnail{
			JobStepId:              uuid.UUID{},
			JobContainerId:         containerId,
			ContainerData:          nil,
			StatusValue:            0,
			ResultId:               nil,
			KubernetesTemplateFile: "",
		},
	}

	container := &JobContainer{
		Id:             containerId,
		Steps:          steps,
		CompletedSteps: 0,
		Status:         JOB_STARTED,
		JobTemplateId:  uuid.UUID{},
	}

	result := container.CompleteStepAndMoveOn()
	if result != steps[1] {
		t.Errorf("Expected step 1, got %s", spew.Sprint(result))
	}

	if container.CompletedSteps != 1 {
		t.Errorf("Expected completed steps to equal 1, got %d", container.CompletedSteps)
	}

	if container.Status != JOB_STARTED {
		t.Errorf("Expected container status %d, got %d", JOB_STARTED, container.Status)
	}

	result2 := container.CompleteStepAndMoveOn()
	if result2 != nil {
		t.Error("Completing last step should return nil, got ", result2)
	}
	if container.Status != JOB_COMPLETED {
		t.Errorf("Expected container status %d, got %d", JOB_COMPLETED, container.Status)
	}
}
