package jobrunner

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"time"
)

type QueueName string

const (
	REQUEST_QUEUE QueueName = "jobrequestqueue"
	RUNNING_QUEUE QueueName = "jobrunningqueue"
)

func keyForJobId(id uuid.UUID) string {
	return fmt.Sprintf("mediaflipper:jobrequest:%s", id.String())
}

func getNextRequestQueueEntry(client *redis.Client) (*JobRunnerRequest, error) {
	return getNextJobRunnerRequest(client, REQUEST_QUEUE)
}

func getNextRunningQueueEntry(client *redis.Client) (*JobRunnerRequest, error) {
	return getNextJobRunnerRequest(client, REQUEST_QUEUE)
}

func getNextJobRunnerRequest(client *redis.Client, queueName QueueName) (*JobRunnerRequest, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	result := client.LPop(jobKey)

	content, getErr := result.Result()
	if getErr != nil {
		log.Print("ERROR: Could not get next item from job queue: ", getErr)
		return nil, getErr
	}

	if content == "" {
		log.Print("DEBUG: no items in queue right now")
		return nil, nil
	}
	var rq JobRunnerRequest
	marshalErr := json.Unmarshal([]byte(content), &rq)
	if marshalErr != nil {
		log.Print("ERROR: Could not decode item from job queue: ", marshalErr)
		//it's already been removed by the LPOP operation
		return nil, marshalErr
	}
	return &rq, nil
}

func pushToRequestQueue(client *redis.Client, item *JobRunnerRequest) error {
	return pushToQueue(client, item, REQUEST_QUEUE)
}

func pushToRunningQueue(client *redis.Client, item *JobRunnerRequest) error {
	return pushToQueue(client, item, RUNNING_QUEUE)
}

func pushToQueue(client *redis.Client, item *JobRunnerRequest, queueName QueueName) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	encodedContent, marshalErr := json.Marshal(item)
	if marshalErr != nil {
		log.Printf("Could not encode content for %s: %s", item, marshalErr)
		return marshalErr
	}

	result := client.RPush(jobKey, string(encodedContent))
	if result.Err() != nil {
		log.Printf("Could not push to list %s: %s", jobKey, result.Err())
		return result.Err()
	}
	return nil
}

func getRequestQueueLength(client *redis.Client) (int64, error) {
	return getQueueLength(client, REQUEST_QUEUE)
}

func getRunningQueueLength(client *redis.Client) (int64, error) {
	return getQueueLength(client, RUNNING_QUEUE)
}

func getQueueLength(client *redis.Client, queueName QueueName) (int64, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	result := client.LLen(jobKey)

	count, err := result.Result()
	if err != nil {
		log.Printf("Could not retrieve queue length for %s: %s", queueName, err)
	}
	return count, err
}

func copyRunningQueueContent(client *redis.Client) (*[]JobRunnerRequest, error) {
	return copyQueueContent(client, RUNNING_QUEUE)
}

/**
download a snapshot of the current queue
*/
func copyQueueContent(client *redis.Client, queueName QueueName) (*[]JobRunnerRequest, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	cmd := client.LRange(jobKey, 0, -1)
	result, err := cmd.Result()

	if err != nil {
		log.Printf("Could not range %s: %s", jobKey, err)
		return nil, err
	}

	rtn := make([]JobRunnerRequest, len(result))
	for i, resultString := range result {
		var rq JobRunnerRequest
		unmarshalEr := json.Unmarshal([]byte(resultString), &rq)
		if unmarshalEr != nil {
			log.Print("ERROR: Corrupted information in ", queueName, " queue: ", unmarshalEr)
			return nil, unmarshalEr
		}
		rtn[i] = rq
	}

	return &rtn, nil
}

/**
remove the item at the given index. you should ensure that you have the lock before doign this!
*/
func removeFromQueue(client *redis.Client, queueName QueueName, idx int64) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	_, err := client.LRem(jobKey, 0, idx).Result()
	if err != nil {
		log.Printf("Could not remove %d from queue %s: %s", idx, queueName, err)
		return err
	}
	return nil
}

/**
check if the given queue lock is set
*/
func checkQueueLock(client *redis.Client, queueName QueueName) (bool, error) {
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
func setQueueLock(client *redis.Client, queueName QueueName) {
	jobKey := fmt.Sprintf("mediaflipper:%s:lock", queueName)

	client.SetXX(jobKey, "set", 2*time.Second)
}

/**
release the given queue lock
*/
func releaseQueueLock(client *redis.Client, queueName QueueName) {
	jobKey := fmt.Sprintf("mediaflipper:%s:lock", queueName)
	client.Del(jobKey)
}
