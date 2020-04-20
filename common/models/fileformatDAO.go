package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

func keyForFileId(id uuid.UUID) string {
	return fmt.Sprintf("mediaflipper:fileformat:%s", id.String())
}

/**
Retrieve the file format data for a given UUID from the datastore. Returns nil, nil if the job does not exist
*/
func GetFileFormat(forId uuid.UUID, redisClient redis.Cmdable) (*FileFormatInfo, error) {
	jobKey := keyForFileId(forId)

	result := redisClient.Get(jobKey)
	content, getErr := result.Result()

	if getErr != nil {
		log.Printf("ERROR fileformatDAO.GetFileFormat could not retrieve file format for %s: %s", forId, getErr)
		return nil, getErr
	}

	var decoded FileFormatInfo
	decodErr := json.Unmarshal([]byte(content), &decoded)
	if decodErr != nil {
		log.Printf("ERROR fileformatDAO.GetFileFormat could not understand content from datastore for %s, removing it: %s", jobKey, decodErr)
		redisClient.Del(jobKey)
		return nil, decodErr
	}
	return &decoded, nil
}

/**
Save the given job object to the datastore. Returns nil if successful, or an error
*/
func PutFileFormat(record *FileFormatInfo, redisClient redis.Cmdable) error {
	jobKey := keyForFileId(record.Id)

	encoded, encodErr := json.Marshal(*record)
	if encodErr != nil {
		log.Print("Could not format data from ", *record, ": ", encodErr)
		return encodErr
	}

	result := redisClient.Set(jobKey, string(encoded), -1)
	if result.Err() != nil {
		log.Printf("ERROR fileformatDAO.PutFileFormat could not save file format entry to datastore: %s", result.Err())
		return result.Err()
	} else {
		return nil
	}
}

func RemoveFileFormat(forId uuid.UUID, redisClient redis.Cmdable) error {
	jobKey := keyForFileId(forId)

	deletedCount, err := redisClient.Del(jobKey).Result()
	log.Printf("INFO fileformatDAO.RemoveFileFormat deleted %d records for file format with id %s", deletedCount, forId)
	return err
}
