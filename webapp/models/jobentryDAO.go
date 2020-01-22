package models

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

func keyForJobId(id uuid.UUID) string {
	return fmt.Sprintf("mediaflipper:job:%s", id.String())
}

/**
Retrieve the job for a given UUID from the datastore. Returns nil, nil if the job does not exist
*/
func GetJobForId(id uuid.UUID, redisClient *redis.Client) (*JobEntry, *error) {
	jobKey := keyForJobId(id)

	result := redisClient.HGetAll(jobKey)
	content, getErr := result.Result()
	if getErr != nil {
		log.Printf("Could not get job for id %s: %s", id.String(), getErr)
		return nil, &getErr
	}

	if len(content) == 0 {
		return nil, nil
	}

	jobEntryPtr, marshalErr := JobEntryFromMap(content)
	if marshalErr != nil {
		log.Printf("Could not marshal data from datastore for job id %s: %s", id.String(), *marshalErr)
		return nil, marshalErr
	}

	return jobEntryPtr, nil
}

/**
Save the given job object to the datastore. Returns nil if successful, or an error
*/
func PutJob(entry *JobEntry, redisClient *redis.Client) error {
	jobKey := keyForJobId(entry.JobId)
	mapData := entry.ToMap()

	pipe := redisClient.Pipeline()
	for k, v := range mapData {
		pipe.HSet(jobKey, k, v)
	}
	_, putErr := pipe.Exec()
	if putErr != nil {
		log.Printf("Could not save job entry %s to datastore: %s", entry.JobId.String(), putErr)
		return putErr
	}
	return nil
}
