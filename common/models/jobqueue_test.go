package models

import (
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"testing"
	"time"
)

func TestJobQueueEntry_Marshal(t *testing.T) {
	newEntry := JobQueueEntry{
		JobId:  uuid.MustParse("814d602e-fbc0-488d-9aa5-0e11556ff846"),
		StepId: uuid.MustParse("987aaeb3-1b5d-4215-a0e3-e3b56017b530"),
		Status: 2,
	}
	result := newEntry.Marshal()
	if result != "814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2" {
		t.Errorf("JobQueueEntry.marshal returned wrong data, expected 814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2 got %s", result)
	}
}

func TestUnmarshalJobQueueEntry(t *testing.T) {
	rawData := "814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2"
	result, err := UnmarshalJobQueueEntry(rawData)
	if err != nil {
		t.Error("UnmarshalJobQueueEntry failed unexpectedly: ", err)
	} else {
		if result.JobId != uuid.MustParse("814d602e-fbc0-488d-9aa5-0e11556ff846") {
			t.Errorf("got incorrect job ID, expected 814d602e-fbc0-488d-9aa5-0e11556ff846 got %s", result.JobId)
		}
		if result.StepId != uuid.MustParse("987aaeb3-1b5d-4215-a0e3-e3b56017b530") {
			t.Errorf("got incorrect step id, expected 987aaeb3-1b5d-4215-a0e3-e3b56017b530 got %s", result.StepId)
		}
		if result.Status != 2 {
			t.Errorf("got incorrect status, expected 2 got %d", result.Status)
		}
	}

	//UnmarshalJobQueueEntry should error if there are the wrong number of sections
	wrongData1 := "814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2|3|4"
	_, shouldErr1 := UnmarshalJobQueueEntry(wrongData1)
	if shouldErr1 == nil {
		t.Errorf("UnmarshalJobQu")
	}
}

func TestGetQueueLength(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	testClient.RPush("mediaflipper:jobrunningqueue", "a", "b", "c", "d", "e", "f", "g")
	result, _ := GetQueueLength(testClient, RUNNING_QUEUE)
	if result != 7 {
		t.Errorf("Got incorrect queue length, expected 7 got %d", result)
	}
}

func TestSnapshotQueue(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	testData := []JobQueueEntry{
		{uuid.UUID{}, uuid.UUID{}, 0},
		{uuid.UUID{}, uuid.UUID{}, 1},
		{uuid.UUID{}, uuid.UUID{}, 2},
		{uuid.UUID{}, uuid.UUID{}, 3},
	}
	encodedData := make([]string, len(testData))
	for i, d := range testData {
		encodedData[i] = d.Marshal()
	}
	testClient.RPush("mediaflipper:jobrunningqueue", encodedData[0], encodedData[1], encodedData[2], encodedData[3])

	result, snapErr := SnapshotQueue(testClient, RUNNING_QUEUE)
	if snapErr != nil {
		t.Error("SnapshotQueue unexpectedly failed: ", snapErr)
	} else {
		if len(result) != 4 {
			t.Errorf("Expected 4 items in the queue, got %d", len(result))
		} else {
			if result[0].Status != JobStatus(0) {
				t.Errorf("first entry had wrong status, expected 0 got %d", result[0].Status)
			}
			if result[1].Status != JobStatus(1) {
				t.Errorf("first entry had wrong status, expected 1 got %d", result[1].Status)
			}
			if result[2].Status != JobStatus(2) {
				t.Errorf("first entry had wrong status, expected 2 got %d", result[2].Status)
			}
			if result[3].Status != JobStatus(3) {
				t.Errorf("first entry had wrong status, expected 3 got %d", result[3].Status)
			}
		}
	}
}

func TestRemoveFromQueue(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	testData := []JobQueueEntry{
		{uuid.UUID{}, uuid.UUID{}, 0},
		{uuid.UUID{}, uuid.UUID{}, 1},
		{uuid.UUID{}, uuid.UUID{}, 2},
		{uuid.UUID{}, uuid.UUID{}, 3},
	}
	encodedData := make([]string, len(testData))
	for i, d := range testData {
		encodedData[i] = d.Marshal()
	}
	testClient.RPush("mediaflipper:jobrunningqueue", encodedData[0], encodedData[1], encodedData[2], encodedData[3])

	remErr := RemoveFromQueue(testClient, RUNNING_QUEUE, testData[2])
	if remErr != nil {
		t.Error("RemoveFromQueue unexpectedly failed: ", remErr)
	} else {
		result, snapErr := SnapshotQueue(testClient, RUNNING_QUEUE)
		if snapErr != nil {
			t.Error("SnapshotQueue unexpectedly failed: ", snapErr)
		} else {
			if len(result) != 3 {
				t.Errorf("Expected 3 items in the queue after deletion, got %d", len(result))
			} else {
				if result[0].Status != JobStatus(0) {
					t.Errorf("first entry had wrong status, expected 0 got %d", result[0].Status)
				}
				if result[1].Status != JobStatus(1) {
					t.Errorf("first entry had wrong status, expected 1 got %d", result[1].Status)
				}
				if result[2].Status != JobStatus(3) {
					t.Errorf("first entry had wrong status, expected 3 got %d", result[2].Status)
				}
			}
		}
	}
}
func TestAddToQueue(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	dbKey := fmt.Sprintf("mediaflipper:%s", RUNNING_QUEUE)

	newEntry := JobQueueEntry{
		JobId:  uuid.MustParse("814d602e-fbc0-488d-9aa5-0e11556ff846"),
		StepId: uuid.MustParse("987aaeb3-1b5d-4215-a0e3-e3b56017b530"),
		Status: 2,
	}

	addErr := AddToQueue(testClient, RUNNING_QUEUE, newEntry)
	if addErr != nil {
		t.Error("AddToQueue failed unexpectedly: ", addErr)
	} else {
		content, _ := testClient.LRange(dbKey, 0, 999).Result()
		if len(content) < 1 {
			t.Error("no data returned when content should have been stored")
		} else {
			if content[0] != "814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2" {
				t.Errorf("returned content incorrect. Expected 814d602e-fbc0-488d-9aa5-0e11556ff846|987aaeb3-1b5d-4215-a0e3-e3b56017b530|2, got '%s'", content[0])
			}
		}
		if len(content) != 1 {
			t.Errorf("got extra items, expected 1 got %d", len(content))
		}
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
