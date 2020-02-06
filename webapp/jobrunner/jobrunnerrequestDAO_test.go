package jobrunner

import (
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	models2 "github.com/guardian/mediaflipper/common/models"
	"log"
	"reflect"
	"testing"
	"time"
)

/**
CopyRunningQueueContent should return a snapshot of the running queue as a list of models.JobStep objects
*/
func TestCopyRunningQueueContent(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	jobStepId := uuid.New()
	testStep1 := models2.JobStepAnalysis{
		JobStepType:            "analysis",
		JobStepId:              jobStepId,
		JobContainerId:         uuid.New(),
		ContainerData:          nil,
		StatusValue:            models2.JOB_STARTED,
		ResultId:               uuid.New(),
		MediaFile:              "path/to/something",
		KubernetesTemplateFile: "mytemplate.yaml",
	}

	testStep1Enc, _ := json.Marshal(testStep1)
	testStep2 := models2.JobStepThumbnail{
		JobStepType: "thumbnail",
	}
	testStep2Enc, _ := json.Marshal(testStep2)

	keyName := fmt.Sprintf("mediaflipper:%s", RUNNING_QUEUE)
	s.Lpush(keyName, string(testStep2Enc))
	s.Lpush(keyName, string(testStep1Enc))

	result, err := copyQueueContent(testClient, RUNNING_QUEUE)
	if err != nil {
		t.Error("copyQueueContent unexpectedly failed: ", err)
		t.FailNow()
	}

	if len(*result) != 2 {
		t.Errorf("expected 2 items from queue, got %d", len(*result))
		t.FailNow()
	}
	decoded := make([]models2.JobStep, len(*result))
	for i, jsonBlob := range *result {
		var rawData map[string]interface{}
		json.Unmarshal([]byte(jsonBlob), &rawData)
		//spew.Dump(rawData)
		var getErr error
		decoded[i], getErr = getJobFromMap(rawData)
		if getErr != nil {
			t.Error("getJobFromMap unexpectedly failed: ", getErr)
		}
	}

	_, isAnalysis := decoded[0].(*models2.JobStepAnalysis)
	if !isAnalysis {
		t.Error("Step 0 was not analysis, got ", reflect.TypeOf(decoded[0]))
	}

	_, isThumb := decoded[1].(*models2.JobStepThumbnail)
	if !isThumb {
		t.Error("Step 1 was not thumbnail, got ", reflect.TypeOf(decoded[0]))
	}
}

func TestQueueLockBasic(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	//non-existing key should always be unlocked
	locked, lockErr := CheckQueueLock(testClient, "testqueue")
	if lockErr != nil {
		t.Error("Expected uninitialised check not to error, got ", lockErr)
	}
	if locked {
		t.Error("Expected uninitialised queue not to be locked")
	}

	//set a lock and check it
	SetQueueLock(testClient, "testqueue")
	shouldBeLocked, lockErr := CheckQueueLock(testClient, "testqueue")
	if lockErr != nil {
		t.Error("Expected locked check not to error, got ", lockErr)
	}
	if !shouldBeLocked {
		t.Error("Expected locked check to be true, got false")
	}

	//clear the lock and check it again
	ReleaseQueueLock(testClient, "testqueue")
	shouldBeUnLocked, lockErr := CheckQueueLock(testClient, "testqueue")
	if lockErr != nil {
		t.Error("Expected locked check not to error, got ", lockErr)
	}
	if shouldBeUnLocked {
		t.Error("Expected unlocked check to be false, got true")
	}
}

func TestWhenQueueAvailable(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	SetQueueLock(testClient, "testqueue")
	unlockTimer := time.NewTimer(500 * time.Millisecond)

	go func() {
		<-unlockTimer.C
		ReleaseQueueLock(testClient, "testqueue")
	}()

	testStartMs := time.Now().UnixNano() / 1000000
	log.Printf("test start time is %d", testStartMs)
	completionChan := make(chan int64)
	callOnRelease := func(failure error) {
		if failure != nil {
			panic(failure)
		}
		completionChan <- time.Now().UnixNano() / 1000000
	}

	WhenQueueAvailable(testClient, "testqueue", callOnRelease, false)
	testDoneMs := <-completionChan
	log.Printf("test finish time is %d", testDoneMs)
	if testDoneMs-testStartMs < 500 {
		t.Errorf("Test completed too quickly, in %dms instead of 500ms. Suggests that the queue wait failed.", testDoneMs-testStartMs)
	}
}
