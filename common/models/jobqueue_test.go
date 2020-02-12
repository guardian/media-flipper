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
