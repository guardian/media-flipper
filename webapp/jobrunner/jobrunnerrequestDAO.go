package jobrunner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
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

func getNextRequestQueueEntry(client *redis.Client) (*models.JobContainer, error) {
	return getNextJobRunnerRequest(client, REQUEST_QUEUE)
}

func getNextJobRunnerRequest(client *redis.Client, queueName QueueName) (*models.JobContainer, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	result := client.LPop(jobKey)

	content, getErr := result.Result()
	if getErr != nil {
		if getErr.Error() == "redis: nil" { //FIXME: there must be a better way of making this check!
			return nil, nil
		}
		log.Print("ERROR: Could not get next item from job queue: ", getErr)
		return nil, getErr
	}

	if content == "" {
		log.Print("DEBUG: no items in queue right now")
		return nil, nil
	}
	var rq models.JobContainer
	log.Printf("DEBUG: Got %s for %s", content, jobKey)

	marshalErr := json.Unmarshal([]byte(content), &rq)
	if marshalErr != nil {
		log.Print("ERROR: Could not decode item from job queue: ", marshalErr)
		//it's already been removed by the LPOP operation
		return nil, marshalErr
	}
	return &rq, nil
}

func pushToRequestQueue(client *redis.Client, item *models.JobContainer) error {
	encodedContent, marshalErr := json.Marshal(*item)
	if marshalErr != nil {
		log.Print("Could not encode content for ", item, ": ", marshalErr)
		return marshalErr
	}

	return pushToQueue(client, encodedContent, REQUEST_QUEUE)
}

func pushToRunningQueue(client *redis.Client, item *models.JobStep) error {
	encodedContent, marshalErr := json.Marshal(*item)
	if marshalErr != nil {
		log.Print("Could not encode content for ", item, ": ", marshalErr)
		return marshalErr
	}

	return pushToQueue(client, encodedContent, RUNNING_QUEUE)
}

func pushToQueue(client *redis.Client, encodedContent []byte, queueName QueueName) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	//log.Printf("DEBUG: Pushed %s to %s", string(encodedContent), jobKey)

	result := client.RPush(jobKey, string(encodedContent))
	if result.Err() != nil {
		log.Printf("Could not push to list %s: %s", jobKey, result.Err())
		return result.Err()
	}
	//log.Printf("DEBUG: pushed %s to %s", item.RequestId, queueName)
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

func getJobFromMap(fromMap map[string]interface{}) (models.JobStep, error) {
	jobType, isStr := fromMap["stepType"].(string)
	if !isStr {
		log.Printf("Could not determine job type, stepType parameter missing or wrong format")
		return nil, errors.New("Could not determine job type")
	}
	switch jobType {
	case "analysis":
		aJobPtr, anErr := models.JobStepAnalysisFromMap(fromMap)
		if anErr == nil {
			log.Printf("Got JobStepAnalysis")
			return aJobPtr, nil
		}
	case "thumbnail":
		tJobPtr, tErr := models.JobStepThumbnailFromMap(fromMap)
		if tErr == nil && tJobPtr.JobStepType == "thumbnail" {
			log.Printf("Got JobStepThumbnail")
			return tJobPtr, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Could not decode to any known job type, got %s", fromMap["stepType"]))
}

func copyRunningQueueContent(client *redis.Client) (*[]models.JobStep, error) {
	result, getErr := copyQueueContent(client, RUNNING_QUEUE)
	if getErr != nil {
		return nil, getErr
	}

	rtn := make([]models.JobStep, len(*result))
	for i, resultString := range *result {
		var rq map[string]interface{}
		//log.Printf("content before unmarshal: %s", resultString)
		unmarshalEr := json.Unmarshal([]byte(resultString), &rq)
		if unmarshalEr != nil {
			log.Print("ERROR: Corrupted information in ", RUNNING_QUEUE, " queue: ", unmarshalEr)
			return nil, unmarshalEr
		}

		step, stepErr := getJobFromMap(rq)
		if stepErr != nil {
			log.Print("ERROR: Corrupted information in ", RUNNING_QUEUE, " queue: ", stepErr)
			return nil, stepErr
		}
		rtn[i] = step
	}

	return &rtn, nil
}

/**
download a snapshot of the current queue
*/
func copyQueueContent(client *redis.Client, queueName QueueName) (*[]string, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	cmd := client.LRange(jobKey, 0, -1)
	result, err := cmd.Result()

	if err != nil {
		log.Printf("Could not range %s: %s", jobKey, err)
		return nil, err
	}
	return &result, nil
}

/**
remove the given item from the given queue.
*/
func removeFromQueue(client *redis.Client, queueName QueueName, entry *models.JobStep) error {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)
	content, _ := json.Marshal(entry)
	//log.Printf("Removing item %s from queue %s", string(content), jobKey)

	_, err := client.LRem(jobKey, 0, string(content)).Result()
	if err != nil {
		log.Printf("Could not remove element from queue %s: %s", queueName, err)
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
