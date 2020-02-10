package initiator

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"io"
	"log"
	"os"
)

type UploadSegment struct {
	ForJob       uuid.UUID `json:"forJob"`
	SegmentIndex int       `json:"index"`
	Filepath     string    `json:"filePath"`
	Checksum     string    `json:"checksum"`
}

/**
checksum the given file and return an UploadSegment object containing the provided information and md5 checksum
*/
func NewUploadSegment(forJob uuid.UUID, segmentIndex int, filePath string) (UploadSegment, error) {
	var rtn = UploadSegment{
		forJob, segmentIndex, filePath, "",
	}

	fp, openErr := os.OpenFile(filePath, os.O_RDONLY, os.FileMode(775))
	if openErr != nil {
		log.Printf("Could not open '%s': %s", filePath, openErr)
		return rtn, openErr
	}
	defer fp.Close()
	hasher := md5.New()

	_, copyErr := io.Copy(hasher, fp)
	if copyErr != nil {
		log.Printf("Could not stream data from %s: %s", filePath, copyErr)
		return rtn, copyErr
	}

	rtn.Checksum = fmt.Sprintf("%x", hasher.Sum(nil)) //nil argument to Sum => use computed value from Write calls
	return rtn, nil
}

/**
write the segment record down to the data store
*/
func (s UploadSegment) Store(redisClient *redis.Client) error {
	dbKey := fmt.Sprintf("mediaflipper:uploadsegment:%s", s.ForJob.String())
	content, err := json.Marshal(s)
	if err != nil {
		return err
	}

	_, setError := redisClient.ZAdd(dbKey, &redis.Z{
		Score:  float64(s.SegmentIndex),
		Member: string(content),
	}).Result()

	return setError
}

/**
delete all segment records for the given job id
*/
func CleanOut(jobId uuid.UUID, redisClient *redis.Client) error {
	dbKey := fmt.Sprintf("mediaflipper:uploadsegment:%s", jobId.String())

	_, err := redisClient.Del(dbKey).Result()
	return err
}
