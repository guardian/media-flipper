package jobrunner

import (
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"github.com/guardian/mediaflipper/webapp/bulkprocessor"
	"testing"
	"time"
)

type TemplateManagerMock struct {
	Test                              *testing.T
	FakeContainer                     *models.JobContainer
	NewJobContainerError              error
	ExpectedNewJobContainerTemplateId uuid.UUID
	ExpectedNewJobContainerItemType   helpers.BulkItemType

	TemplateDefinitions []models.JobTemplateDefinition
}

func (m TemplateManagerMock) NewJobContainer(templateId uuid.UUID, itemType helpers.BulkItemType) (*models.JobContainer, error) {
	//if templateId != m.ExpectedNewJobContainerTemplateId {
	//	m.Test.Errorf("NewJobContainer called with incorrect templateId, expected %s got %s", m.ExpectedNewJobContainerTemplateId, templateId)
	//}
	//if itemType != m.ExpectedNewJobContainerItemType {
	//	m.Test.Errorf("NewJobContainer called with incorrect itemType, expected %s got %s", m.ExpectedNewJobContainerItemType, itemType)
	//}
	if m.NewJobContainerError != nil {
		return nil, m.NewJobContainerError
	}
	return m.FakeContainer, nil
}

func (m TemplateManagerMock) ListTemplates() []models.JobTemplateDefinition {
	return m.TemplateDefinitions
}

func (m TemplateManagerMock) GetJob(jobId uuid.UUID) (models.JobTemplateDefinition, bool) {
	return models.JobTemplateDefinition{}, false
}

type JobRunnerMockRealEnqueue struct {
	Test                    *testing.T
	AddJobExpectedContainer models.JobContainer
	AddJobReturnError       error

	AddedContainers []*models.JobContainer
	WrapperRunner   *JobRunner
}

func (m *JobRunnerMockRealEnqueue) AddJob(container *models.JobContainer) error {
	m.AddedContainers = append(m.AddedContainers, container)
	if m.AddJobReturnError != nil {
		return m.AddJobReturnError
	}
	return nil
}

func (m *JobRunnerMockRealEnqueue) EnqueueContentsAsync(redisClient redis.Cmdable, templateManager models.TemplateManagerIF, l *bulkprocessor.BulkListImpl, testRunner JobRunnerIF) chan error {
	return m.WrapperRunner.EnqueueContentsAsync(redisClient, templateManager, l, nil, bulkprocessor.ITEM_STATE_NOT_QUEUED, testRunner)
}

func (m *JobRunnerMockRealEnqueue) clearCompletedTick() {

}

/**
EnqueueContentsAsync should:
 - create a job from template for each item of the bulk
 - add that job to the provided runner
*/
func TestJobRunner_EnqueueContentsAsync(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	bulkId := uuid.MustParse("3479D940-0285-45BB-AE26-AEB69F811145")
	itemId := uuid.MustParse("BF7E9BF6-9F5C-463D-B8A2-918EB4A6409D")

	jobId := uuid.MustParse("8020F41F-4E46-489D-BB27-77BDCB27DC3B")
	nowTime := time.Now()
	mockedJobContainer := &models.JobContainer{
		Id:                jobId,
		Steps:             []models.JobStep{},
		CompletedSteps:    0,
		Status:            0,
		JobTemplateId:     uuid.UUID{},
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         &nowTime,
		EndTime:           nil,
		AssociatedBulk: &models.BulkAssociation{
			Item: itemId,
			List: bulkId,
		},
		ItemType:          "",
		ThumbnailId:       nil,
		TranscodedMediaId: nil,
	}

	templateMgr := TemplateManagerMock{
		Test:                              t,
		FakeContainer:                     mockedJobContainer,
		NewJobContainerError:              nil,
		ExpectedNewJobContainerTemplateId: uuid.UUID{},
		ExpectedNewJobContainerItemType:   "",
		TemplateDefinitions:               nil,
	}

	//realRunner := NewJobRunner(testClient, nil, nil, 1, false)
	realRunner := JobRunner{
		redisClient:     testClient,
		jobClient:       nil,
		serviceClient:   nil,
		podClient:       nil,
		shutdownChan:    nil,
		queuePollTicker: nil,
		templateMgr:     nil,
		maxJobs:         1,
		bulkListDAO:     bulkprocessor.BulkListDAOImpl{},
	}

	runner := &JobRunnerMockRealEnqueue{
		Test:                    t,
		AddJobExpectedContainer: *mockedJobContainer,
		AddJobReturnError:       nil,
		WrapperRunner:           &realRunner,
	}

	bulk := bulkprocessor.BulkListImpl{
		BulkListId:      bulkId,
		CreationTime:    time.Time{},
		NickName:        "",
		VideoTemplateId: uuid.UUID{},
		AudioTemplateId: uuid.UUID{},
		ImageTemplateId: uuid.UUID{},
		BulkListDAO:     bulkprocessor.BulkListDAOImpl{},
	}

	testRecord := bulkprocessor.BulkItemImpl{
		Id:         itemId,
		BulkListId: bulkId,
		SourcePath: "path/to/videofile",
		Priority:   0,
		State:      bulkprocessor.ITEM_STATE_NOT_QUEUED,
		Type:       helpers.ITEM_TYPE_VIDEO,
	}
	addErr := bulk.AddRecord(&testRecord, testClient)
	if addErr != nil {
		t.Error("AddRecord failed unexpectedly: ", addErr)
		t.FailNow()
	}

	errorChan := runner.EnqueueContentsAsync(testClient, templateMgr, &bulk, runner)
	actions, actsErr := bulk.GetActionsRunning(testClient)
	if actsErr != nil {
		t.Error("GetActionsRunning failed unexpectedly: ", actsErr)
	} else {
		if len(actions) != 1 {
			t.Errorf("Got wrong number of current actions, expected 1 got %d", len(actions))
		} else {
			if actions[0] != bulkprocessor.JOBS_QUEUEING {
				t.Errorf("Got wrong current action, expected %s got %s", bulkprocessor.JOBS_QUEUEING, actions[0])
			}
		}
	}

	receivedErr := <-errorChan //wait for async to complete

	postActs, postActsErr := bulk.GetActionsRunning(testClient)
	if postActsErr != nil {
		t.Error("GetActionsRunning failed unexpectedly once the async finished: ", postActsErr)
	} else {
		if len(postActs) != 0 {
			t.Errorf("Action was not cleared")
		}
	}
	if receivedErr != nil {
		t.Errorf("EnqueueContentsAsync failed unexpectedly: %s", receivedErr)
	}
	if len(runner.AddedContainers) != 1 {
		t.Errorf("Got wrong number of jobs added to the runner, expected 1 got %d", len(runner.AddedContainers))
	} else {
		if runner.AddedContainers[0] != mockedJobContainer {
			t.Errorf("Got wrong container added to the runner, expected %s got %s",
				spew.Sdump(mockedJobContainer), spew.Sdump(runner.AddedContainers[0]))
		}
	}
}
