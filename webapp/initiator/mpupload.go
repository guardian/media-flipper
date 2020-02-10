package initiator

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
)

//describe the interface to the class, this allows it to be mocked in tests
type MPUploadIf interface {
	UploadedParts(client *redis.Client) (int32, error)
	IsCompleted(client *redis.Client) (bool, error)
	CompoundChecksum(client *redis.Client) (string, error)
}

//concrete implementation
type MPUpload struct {
	ForJob        uuid.UUID `json:"forJob"`
	ExpectedParts int32     `json:"expectedParts"`
}

/**
returns the number of parts that are uploaded and available
*/
func (u MPUpload) UploadedParts(redisClient *redis.Client) (int32, error) {
	dbKey := fmt.Sprintf("mediaflipper:uploadsegment:%s", u.ForJob.String())

	count, err := redisClient.ZCard(dbKey).Result()
	if err != nil {
		return 0, err
	} else {
		return int32(count), nil
	}
}

/**
returns a boolean indicating whether we have all of the parts that we are expecting
*/
func (u MPUpload) IsCompleted(redisClient *redis.Client) (bool, error) {
	uploadedParts, getErr := u.UploadedParts(redisClient)
	if getErr != nil {
		return false, getErr
	} else {
		return u.ExpectedParts > uploadedParts, nil
	}
}
