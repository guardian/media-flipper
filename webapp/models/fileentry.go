package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"log"
	"os"
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
	MimeType       string    `json:"mimeType"`
	Size           int64     `json:"size"`
}

func NewFileEntry(forPath string, jobContainerId uuid.UUID, fileType FileType) (FileEntry, error) {
	statInfo, statErr := os.Stat(forPath)
	if statErr != nil {
		return FileEntry{}, statErr
	}

	fileTypeInfo, ftErr := filetype.MatchFile(forPath)
	var mimeType string
	if ftErr != nil {
		log.Printf("Could not determine type for %s: %s", forPath, ftErr)
		mimeType = "application/octet-stream"
	} else {
		if fileTypeInfo.MIME.Value != "" {
			mimeType = fileTypeInfo.MIME.Value
		} else {
			log.Printf("Got no MIME type for %s", forPath)
			mimeType = "application/octet-stream"
		}
	}

	return FileEntry{
		Id:             uuid.New(),
		ServerPath:     forPath,
		JobContainerId: jobContainerId,
		FileType:       fileType,
		MimeType:       mimeType,
		Size:           statInfo.Size(),
	}, nil
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
	dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", forId)
	rawContent, getErr := redisClient.Get(dbKey).Result()

	if getErr != nil {
		log.Printf("Could not retrieve file entry for %s: %s", forId.String(), getErr)
		return nil, getErr
	}

	var ent FileEntry
	marshalErr := json.Unmarshal([]byte(rawContent), &ent)
	if marshalErr != nil {
		log.Printf("Corrupted information in the datastore for %s: %s", dbKey, marshalErr)
		return nil, marshalErr
	}

	return &ent, nil
}

/**
retrieves a list of the file IDs for the given job ID. Called internally by FilesForJobContainer.
returns a pointer to a list of UUIDs on success or an error on error.
*/
func FileIdsForJobContainer(jobMasterId uuid.UUID, client redis.Client) (*[]uuid.UUID, error) {
	indexKey := fmt.Sprintf("mediaflipper:jobfile:%s", jobMasterId)

	indexResult, indexErr := client.HGetAll(indexKey).Result()
	if indexErr != nil {
		log.Printf("Could not retrieve index entries for job %s: %s", jobMasterId.String(), indexErr)
		return nil, indexErr
	}

	uuidList := make([]uuid.UUID, len(indexResult))
	i := 0
	for _, idString := range indexResult {
		var uuidErr error
		uuidList[i], uuidErr = uuid.Parse(idString)
		if uuidErr != nil {
			log.Printf("Could not parse uuid from %s: %s", idString, uuidErr)
		}
	}
	return &uuidList, nil
}

/**
retrieves a list of FileEntries for the given job ID
*/
func FilesForJobContainer(jobMasterId uuid.UUID, client redis.Client) (*[]FileEntry, error) {
	fileIdList, idGetErr := FileIdsForJobContainer(jobMasterId, client)
	if idGetErr != nil {
		return nil, idGetErr
	}

	pipe := client.Pipeline()
	defer pipe.Close()

	entryList := make([]FileEntry, len(*fileIdList))
	for _, fileId := range *fileIdList {
		dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", fileId)
		pipe.Get(dbKey)
	}
	results, err := pipe.Exec()
	if err != nil {
		log.Printf("Pipelined GET failed: %s", err)
		return nil, err
	}

	failedCtr := 0
	for i, result := range results {
		s := result.(*redis.StringCmd)
		getResult := s.Val()
		marshalErr := json.Unmarshal([]byte(getResult), &entryList[i])
		if marshalErr != nil {
			log.Printf("Invalid data for file entry %s: %s. Offending data was: %s", s.Name(), marshalErr, s.Val())
			failedCtr += 1
		}
	}
	if failedCtr == len(results) {
		//everything failed
		return nil, errors.New("no data was unmarshalled, see logs for details")
	}
	return &entryList, nil
}
