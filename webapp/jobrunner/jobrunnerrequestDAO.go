package jobrunner

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"log"
)

func getNextRequestQueueEntry(client *redis.Client) (*models.JobContainer, error) {
	return getNextJobRunnerRequest(client, models.REQUEST_QUEUE)
}

func getNextJobRunnerRequest(client *redis.Client, queueName models.QueueName) (*models.JobContainer, error) {
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

	return pushToQueue(client, encodedContent, models.REQUEST_QUEUE)
}

func pushToQueue(client *redis.Client, encodedContent []byte, queueName models.QueueName) error {
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

/**
download a snapshot of the current queue. it's a good idea to assert the queue lock before taking the snapshot
and release it when the processing is done, so that the queue content remains valid.
*/
func copyQueueContent(client redis.Cmdable, queueName models.QueueName) ([]string, error) {
	jobKey := fmt.Sprintf("mediaflipper:%s", queueName)

	cmd := client.LRange(jobKey, 0, -1)
	result, err := cmd.Result()

	if err != nil {
		log.Printf("Could not range %s: %s", jobKey, err)
		return nil, err
	}
	return result, nil
}
