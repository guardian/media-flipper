package models

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"strconv"
	"strings"
	"time"
)

type QueueName string

const (
	REQUEST_QUEUE QueueName = "jobrequestqueue"
	RUNNING_QUEUE QueueName = "jobrunningqueue"
)

/** -----------------
queue entry data
----------------
*/
type JobQueueEntry struct {
	JobId  uuid.UUID
	StepId uuid.UUID
	Status JobStatus
}

func (j JobQueueEntry) Marshal() string {
	return j.JobId.String() + "|" + j.StepId.String() + "|" + strconv.FormatInt(int64(j.Status), 10)
}

func UnmarshalJobQueueEntry(from string) (JobQueueEntry, error) {
	parts := strings.Split(from, "|")
	if len(parts) != 3 {
		return JobQueueEntry{}, errors.New("incorrect data, did not have 3 sections")
	}
	jobId, jobIdErr := uuid.Parse(parts[0])
	if jobIdErr != nil {
		return JobQueueEntry{}, jobIdErr
	}
	stepId, stepIdErr := uuid.Parse(parts[1])
	if stepIdErr != nil {
		return JobQueueEntry{}, stepIdErr
	}
	jobStatus, statusErr := strconv.ParseInt(parts[2], 10, 8)
	if statusErr != nil {
		return JobQueueEntry{}, statusErr
	}

	return JobQueueEntry{
		JobId:  jobId,
		StepId: stepId,
		Status: JobStatus(jobStatus),
	}, nil
}

/** -----------------
queue manipulation
----------------
*/
func GetQueueLength(client redis.Cmdable, queueName QueueName) (int64, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	result := client.LLen(jobKey)

	count, err := result.Result()
	if err != nil {
		log.Printf("Could not retrieve queue length for %s: %s", queueName, err)
	}
	return count, err
}

/**
get a 'snapshot' of the queue state at this moment in time.
it is recommended to acquire the queue lock first and not release it until done,
so that the snapshot
*/
func SnapshotQueue(client redis.Cmdable, queueName QueueName) ([]JobQueueEntry, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	rawData, err := client.LRange(jobKey, 0, -1).Result()

	if err != nil {
		log.Printf("Could not range %s: %s", jobKey, err)
		return nil, err
	}

	result := make([]JobQueueEntry, len(rawData))
	for i, rawEntry := range rawData {
		ent, parseErr := UnmarshalJobQueueEntry(rawEntry)
		if parseErr != nil {
			log.Printf("ERROR: Bad data in the %s queue: %s. Offending data was %s.", jobKey, parseErr, rawEntry)
			return nil, parseErr
		}
		result[i] = ent
	}
	return result, nil
}

func RemoveFromQueue(client redis.Cmdable, queueName QueueName, entry JobQueueEntry) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	removed, err := client.LRem(jobKey, 0, entry.Marshal()).Result()
	if err != nil {
		log.Printf("Could not remove %s from %s: %s", entry.Marshal(), jobKey, err)
		return err
	}
	if removed == 0 {
		log.Printf("WARNING: Could not find item %s to remove from queue %s", entry.Marshal(), jobKey)
		return errors.New("could not find item to remove from queue")
	}
	return nil
}

func AddToQueue(client redis.Cmdable, queueName QueueName, entry JobQueueEntry) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	_, err := client.RPush(jobKey, entry.Marshal()).Result()
	return err
}

/** -----------------
locking functions
----------------
*/

/**
check if the given queue lock is set
*/
func CheckQueueLock(client *redis.Client, queueName QueueName) (bool, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s:lock", queueName)

	result, err := client.Exists(jobKey).Result()
	if err != nil {
		log.Printf("Could not check lock for %s: %s", jobKey, err)
		return true, err
	}
	if result > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

/**
set the given queue lock
*/
func SetQueueLock(client *redis.Client, queueName QueueName) {
	jobKey := fmt.Sprintf("mediaflipper:%s:lock", queueName)

	client.Set(jobKey, "set", 2*time.Second)
}

/**
release the given queue lock
*/
func ReleaseQueueLock(client *redis.Client, queueName QueueName) {
	jobKey := fmt.Sprintf("mediaflipper:%s:lock", queueName)
	client.Del(jobKey)
}

/*
block until the given queue lock is available or the timeout occurs
*/
func WaitForQueueLock(client *redis.Client, queueName QueueName, timeout time.Duration) error {
	timeoutTimer := time.NewTicker(timeout)
	clearedChannel := make(chan error)

	go func() {
		for {
			locked, checkErr := CheckQueueLock(client, queueName)
			if checkErr != nil {
				clearedChannel <- checkErr
			} else {
				if locked {
					time.Sleep(50 * time.Millisecond)
				} else {
					clearedChannel <- nil
				}
			}
		}
	}()

	select {
	case <-timeoutTimer.C:
		return errors.New(fmt.Sprintf("Timed out waiting for lock on %s", queueName))
	case checkErr := <-clearedChannel:
		return checkErr
	}
}

type QueueLockCallback func(error)

/*
call the given callback (in a subthread) as soon as the queue becomes unlocked.
optionally, assert the queue lock by calling SetQueueLock/ReleaseQueueLock either side of the callback
remember that the callback is in a background goroutine, concurrency warnings apply
*/
func WhenQueueAvailable(client *redis.Client, queueName QueueName, callback QueueLockCallback, assertingQueue bool) {
	intervalTicker := time.NewTicker(50 * time.Millisecond)
	go func() {
		for {
			select {
			case <-intervalTicker.C:
				locked, checkErr := CheckQueueLock(client, queueName)
				if checkErr != nil {
					log.Printf("ERROR: Could not check lock for %s: %s", queueName, checkErr)
					intervalTicker.Stop()
					callback(checkErr)
					return
				}
				if !locked {
					intervalTicker.Stop()
					if assertingQueue {
						SetQueueLock(client, queueName)
						defer ReleaseQueueLock(client, queueName)
					}
					callback(nil)
					return
				}
			}
		}
	}()
}
