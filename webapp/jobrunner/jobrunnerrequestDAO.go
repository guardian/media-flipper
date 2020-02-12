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

	jobKey := fmt.Sprintf("mediaflipper:%s", models.REQUEST_QUEUE)

	result := client.RPush(jobKey, string(encodedContent))
	if result.Err() != nil {
		log.Printf("Could not push to list %s: %s", jobKey, result.Err())
		return result.Err()
	}
	return nil
}
