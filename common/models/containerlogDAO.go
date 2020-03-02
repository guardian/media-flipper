package models

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"io"
	"strings"
)

/**
removes the logs for the given container from the datastore.
returns an error if it could not be done or nil if it could
*/
func RemoveContainerLog(forStepId uuid.UUID, redisClient redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:containerlog:%s", forStepId)

	_, err := redisClient.Del(dbKey).Result()
	return err
}

func GetContainerLogContent(forStepId uuid.UUID, redisClient redis.Cmdable) (string, error) {
	dbKey := fmt.Sprintf("mediaflipper:containerlog:%s", forStepId)

	return redisClient.Get(dbKey).Result()
}

func GetContainerLogContentStream(forStepId uuid.UUID, client redis.Cmdable) (io.Reader, error) {
	str, err := GetContainerLogContent(forStepId, client)

	if err != nil {
		return nil, err
	}

	return strings.NewReader(str), nil
}
