package models

import (
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
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

func TestIndexLuaConcat(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	jobId := uuid.MustParse("492B3831-E7B1-48A0-8C1B-5A8B4A4D8EF3")
	bulkAssociation := BulkAssociation{
		Item: uuid.MustParse("2CE71052-13C2-4237-A089-5345AC2C7CE5"),
		List: uuid.UUID{},
	}

	ent := JobContainer{
		Id:                jobId,
		Steps:             nil,
		CompletedSteps:    0,
		Status:            0,
		JobTemplateId:     uuid.UUID{},
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         nil,
		EndTime:           nil,
		AssociatedBulk:    &bulkAssociation,
		ItemType:          "",
		ThumbnailId:       nil,
		TranscodedMediaId: nil,
		OutputPath:        "",
	}

	//indexLuaConcat should write a new value
	testErr := indexLuaConcat(&ent, testClient)
	if testErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", testErr)
	}
	result := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	expectedValue := "|492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3"
	if result != expectedValue {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", expectedValue, result)
	}

	//indexLuaConcat should append to an existing value
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "|someexistingvalue")

	existingTestErr := indexLuaConcat(&ent, testClient)
	if existingTestErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", testErr)
	}
	existingTestResult := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	existingTestExpected := "|someexistingvalue|492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3"
	if existingTestResult != existingTestExpected {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", existingTestExpected, existingTestResult)
	}
}

func TestIndexLuaRemove(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	jobId := uuid.MustParse("492B3831-E7B1-48A0-8C1B-5A8B4A4D8EF3")
	bulkAssociation := BulkAssociation{
		Item: uuid.MustParse("2CE71052-13C2-4237-A089-5345AC2C7CE5"),
		List: uuid.UUID{},
	}

	ent := JobContainer{
		Id:                jobId,
		Steps:             nil,
		CompletedSteps:    0,
		Status:            0,
		JobTemplateId:     uuid.UUID{},
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         nil,
		EndTime:           nil,
		AssociatedBulk:    &bulkAssociation,
		ItemType:          "",
		ThumbnailId:       nil,
		TranscodedMediaId: nil,
		OutputPath:        "",
	}

	//indexLuaRemove should remove an existing value
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "|someexistingvalue|492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3")

	testErr := indexLuaRemove(ent.Id, ent.AssociatedBulk, testClient)
	if testErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", testErr)
	}
	result := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	expectedValue := "someexistingvalue"
	if result != expectedValue {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", expectedValue, result)
	}

	//indexLuaRemove should pass through if value does not exist
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "someexistingvalue|a8e41b0e-a49f-41c4-b078-fed9a33ac618|0397e3a2-f192-458b-82df-3016ce4d4310")

	notExistErr := indexLuaRemove(ent.Id, ent.AssociatedBulk, testClient)
	if notExistErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", notExistErr)
	}
	notExistResult := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	notExistExpected := "someexistingvalue|a8e41b0e-a49f-41c4-b078-fed9a33ac618|0397e3a2-f192-458b-82df-3016ce4d4310"
	if notExistResult != notExistExpected {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", notExistExpected, notExistResult)
	}

	//indexLuaRemove should remove multiple instances of the given value
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3|someexistingvalue|a8e41b0e-a49f-41c4-b078-fed9a33ac618|492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3|0397e3a2-f192-458b-82df-3016ce4d4310")

	multiErr := indexLuaRemove(ent.Id, ent.AssociatedBulk, testClient)
	if multiErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", multiErr)
	}
	multiResult := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	multiExpected := "someexistingvalue|a8e41b0e-a49f-41c4-b078-fed9a33ac618|0397e3a2-f192-458b-82df-3016ce4d4310"
	if multiResult != multiExpected {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", multiExpected, multiResult)
	}

	//indexLuaRemove should completely remove hash key if it is empty
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3")
	s.HSet(JOBIDX_BULKITEMASSOCIATION, "anotherkey", "anothervalue")
	finalErr := indexLuaRemove(ent.Id, ent.AssociatedBulk, testClient)
	if finalErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", finalErr)
	}
	keys, getErr := s.HKeys(JOBIDX_BULKITEMASSOCIATION)
	if getErr != nil {
		t.Errorf("could not test keys: %s", getErr)
	} else {
		if len(keys) != 1 {
			t.Errorf("expected key to have been deleted but got %s", spew.Sdump(keys))
		}
	}

	//indexLuaRemove should do nothing if associatedBulk is nil
	s.HSet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String(), "492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3")
	s.HSet(JOBIDX_BULKITEMASSOCIATION, "anotherkey", "anothervalue")
	nopErr := indexLuaRemove(ent.Id, nil, testClient)
	if nopErr != nil {
		t.Errorf("indexLuaConcat failed unexpectedly: %s", nopErr)
	}
	nopResult := s.HGet(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
	nopExpected := "492b3831-e7b1-48a0-8c1b-5a8b4a4d8ef3"
	if nopResult != nopExpected {
		t.Errorf("indexLuaConcat did not write correct value, expected %s got %s", nopExpected, nopResult)
	}
}
