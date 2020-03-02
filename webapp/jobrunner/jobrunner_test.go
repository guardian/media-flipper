package jobrunner

import (
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
	v1 "k8s.io/api/batch/v1"
	"log"
	"testing"
	"time"
)

func setupFakeQueueEntry(redisclient redis.Cmdable) (uuid.UUID, uuid.UUID) {
	jobId := uuid.MustParse("EB010E87-845B-4259-93FD-BAF6BA796672")
	stepId := uuid.MustParse("47E0BCD5-E028-413A-BCF8-64EE10F93DB0")

	fakeQueueEntry := models.JobQueueEntry{
		JobId:  jobId,
		StepId: stepId,
		Status: 0,
	}

	models.AddToQueue(redisclient, models.RUNNING_QUEUE, fakeQueueEntry)
	return jobId, stepId
}

/**
if no runner could be found, clearcompletedtick should:
 - remove the item from the running queue
 - set the job step state to "lost"
*/
func TestJobRunner_clearCompletedTick_notfound(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	jobId, stepId := setupFakeQueueEntry(testClient)
	nowTime := time.Now()

	testJob := models.JobContainer{
		Id: jobId,
		Steps: []models.JobStep{
			models.JobStepAnalysis{
				JobStepType:            "analysis",
				JobStepId:              stepId,
				JobContainerId:         jobId,
				ContainerData:          nil,
				StatusValue:            models.JOB_STARTED,
				ResultId:               uuid.UUID{},
				LastError:              "",
				MediaFile:              "",
				KubernetesTemplateFile: "",
				StartTime:              &nowTime, //if it's not set then you get a segfault when indexing
				EndTime:                nil,
				ItemType:               "",
			},
		},
		CompletedSteps:    0,
		Status:            models.JOB_STARTED,
		JobTemplateId:     uuid.UUID{},
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         &nowTime,
		EndTime:           nil,
		AssociatedBulk:    nil,
		ItemType:          "",
		ThumbnailId:       nil,
		TranscodedMediaId: nil,
		OutputPath:        "",
	}
	storErr := testJob.Store(testClient)
	if storErr != nil {
		t.Error("could not store test job: ", storErr)
		t.FailNow()
	}

	mockJobClient := JobInterfaceMock{
		ListResult: &v1.JobList{
			Items: []v1.Job{},
		},
	}

	runner := JobRunner{redisClient: testClient,
		jobClient:       &mockJobClient,
		shutdownChan:    nil,
		queuePollTicker: nil,
		templateMgr:     nil,
		maxJobs:         10}

	runner.clearCompletedTick()

	queueLen, qlErr := models.GetQueueLength(testClient, models.RUNNING_QUEUE)
	if qlErr != nil {
		t.Error("GetQueueLength failed with error: ", qlErr)
	} else {
		if queueLen != 0 {
			qContent, _ := models.SnapshotQueue(testClient, models.RUNNING_QUEUE)
			for _, item := range qContent {
				log.Printf("\t%s", spew.Sdump(item))
			}
			t.Error("Expected running queue to be empty but it contained the above items")
		}
	}

	if mockJobClient.ListCalledWith == nil {
		t.Error("List was not called with arguments, this would have targeted every job in the namespace!")
	} else {
		if mockJobClient.ListCalledWith.LabelSelector != fmt.Sprintf("mediaflipper.jobStepId=%s", stepId) {
			t.Errorf("List was called with incorrect label selector, expected %s got %s",
				fmt.Sprintf("mediaflipper.jobStepId=%s", stepId), mockJobClient.ListCalledWith.LabelSelector)
		}
	}

	updatedJob, retrieveErr := models.JobContainerForId(jobId, testClient)
	if retrieveErr != nil {
		t.Errorf("could not retrieve updated job info: %s", retrieveErr)
	} else {
		if updatedJob.Steps[0].StepId() != stepId {
			t.Errorf("retrieve job had incorrect step id! expected %s got %s", stepId, updatedJob.Steps[0].StepId())
		}
		if updatedJob.Steps[0].Status() != models.JOB_LOST {
			t.Errorf("updated job step had incorrect status. Expected %d (JOB_LOST), got %d", models.JOB_LOST, updatedJob.Steps[0].Status())
		}
		if updatedJob.Status != models.JOB_LOST {
			t.Errorf("updated job container had incorrect status. Expected %d (JOB_LOST), got %d", models.JOB_LOST, updatedJob.Status)
		}
	}
}
