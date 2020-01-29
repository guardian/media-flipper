package models

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

type FileType string

const (
	TYPE_THUMBNAIL FileType = "thumbnail"
	TYPE_ORIGINAL  FileType = "original"
	TYPE_TRANSCODE FileType = "transcode"
	TYPE_SIDECAR   FileType = "sidecar"
)

type FileEntry struct {
	Id             uuid.UUID `json:"fileId"`
	ServerPath     string
	JobContainerId uuid.UUID `json:"forJob"`
	FileType       FileType  `json:"type"`
}

func (f FileEntry) Store(redisClient *redis.Client) error {
	dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", f.Id)

	content, marshalErr := json.Marshal(f)
	if marshalErr != nil {
		log.Printf("Could not format content: %s. Offending data was %s", marshalErr, spew.Sdump(f))
		return marshalErr
	}

	_, setErr := redisClient.Set(dbKey, string(content), -1).Result()
	if setErr != nil {
		log.Printf("Could not store content: %s", setErr)
		return setErr
	}

	//also set an "index", giving the id of the item in a hash that is bound to the container id.
	//this makes it simple to retrieve the files associated with a given job
	indexKey := fmt.Sprintf("mediaflipper:jobfile:%s", f.JobContainerId)
	_, indexSetErr := redisClient.HSet(indexKey, string(f.FileType), f.Id.String()).Result()
	if indexSetErr != nil {
		log.Printf("Could not update index record: %s", indexSetErr)
		return indexSetErr
	}
	return nil
}

/**
retrieves a FileEntry for the given file entry ID. returns nil if there is nothing present, or an error if the query fails
*/
func FileEntryForId(forId uuid.UUID, redisClient *redis.Client) (*FileEntry, error) {

}

func FileIdsForJobContainer(jobMasterId uuid.UUID, client redis.Client) (*[]uuid.UUID, error) {

}

/**
retrieves a list of FileEntries for the given job ID
*/
func FilesForJobContainer(jobMasterId uuid.UUID, client redis.Client) (*[]FileEntry, error) {

}
