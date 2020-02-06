package models

import (
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"os"
	"reflect"
	"testing"
)

/**
NewFileEntry should create a new file object if the file exists, auto-detecting size and MIME type.
If the file does not exist an error should be returned
*/
func TestNewFileEntry(t *testing.T) {
	fakeId := uuid.New()
	existingFile, err := NewFileEntry("fileentry.go", fakeId, TYPE_ORIGINAL)

	if err != nil {
		t.Error("File entry for existing file failed: ", err)
	} else {
		if existingFile.Id == existingFile.JobContainerId {
			t.Error("File entry Id should not be the same as the job container ID")
		}
		if existingFile.JobContainerId != fakeId {
			t.Errorf("Expected job container id %s, got %s", fakeId, existingFile.JobContainerId)
		}
		if existingFile.FileType != TYPE_ORIGINAL {
			t.Errorf("File entry picked up wrong type, expected %s got %s", TYPE_ORIGINAL, existingFile.FileType)
		}
		if existingFile.Size < 2048 {
			t.Errorf("Detected file size was suspiciously short. Got %d, expected >2048", existingFile.Size)
		}
		if existingFile.MimeType != "application/octet-stream" {
			t.Errorf("Expected MimeType application/octet-stream, got %s", existingFile.MimeType)
		}
		if existingFile.ServerPath != "fileentry.go" {
			t.Errorf("Got incorrect filepath %s", existingFile.ServerPath)
		}
		spew.Dump(existingFile)
	}

	_, notExistError := NewFileEntry("gfkjdfgjhkdfsgjgkhsdsgdjkg", fakeId, TYPE_ORIGINAL)
	if notExistError == nil {
		t.Error("Initiating with a non-existing file did not fail")
	} else {
		if !os.IsNotExist(notExistError) {
			t.Errorf("Should have got a 'not exist' error but got %s", reflect.TypeOf(notExistError))
		}
	}
}

/**
FileEntry::Store should write the contents down to the datastore as an independent key and also write an index
record containing the ID in a hash against the job ID
*/
func TestFileEntry_Store(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	fileId := uuid.New()
	jobId := uuid.New()
	ent := FileEntry{
		Id:             fileId,
		ServerPath:     "path/to/some/file",
		JobContainerId: jobId,
		FileType:       TYPE_ORIGINAL,
		MimeType:       "video/mp4",
		Size:           123456,
	}

	storErr := ent.Store(testClient)
	if storErr != nil {
		t.Errorf("Got an error when saving: %s", storErr)
	} else {
		dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", fileId)
		stringData, _ := s.Get(dbKey)
		if stringData == "" {
			t.Error("Got no data for stored key")
		} else {
			var storedEnt FileEntry
			parseErr := json.Unmarshal([]byte(stringData), &storedEnt)
			if parseErr != nil {
				t.Errorf("Unexpected data stored, could not parse: %s %s", parseErr, stringData)
			} else {
				if storedEnt != ent {
					t.Errorf("Stored data did not match original data")
					spew.Dump(ent)
					spew.Dump(storedEnt)
				}
			}
		}

		indexKey := fmt.Sprintf("mediaflipper:jobfile:%s", jobId)
		indexJobKeys, _ := s.HKeys(indexKey)
		if len(indexJobKeys) != 1 {
			t.Errorf("Unexpected number of keys for jobfile index. Expected %d, got %d", 1, len(indexJobKeys))
		}
		if indexJobKeys[0] != string(TYPE_ORIGINAL) {
			t.Errorf("Incorrect type for first key: expected %s, got %s", TYPE_ORIGINAL, indexJobKeys[0])
		} else {
			indexData := s.HGet(indexKey, string(TYPE_ORIGINAL))
			if indexData != fileId.String() {
				t.Errorf("Got wrong ID for TYPE_ORIGINAL. Expected %s, got %s", fileId.String(), indexData)
			}
		}
	}
}

/**
FileEntryForId should retrieve a pre-existing entry from the datastore
*/
func TestFileEntryForId(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	fileId := uuid.New()
	jobId := uuid.New()
	ent := FileEntry{
		Id:             fileId,
		ServerPath:     "path/to/some/file",
		JobContainerId: jobId,
		FileType:       TYPE_ORIGINAL,
		MimeType:       "video/mp4",
		Size:           123456,
	}
	dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", fileId)
	jsonString, _ := json.Marshal(&ent)
	setErr := s.Set(dbKey, string(jsonString))
	if setErr != nil {
		t.Errorf("TEST ERROR: could not write test record: %s", setErr)
		t.FailNow()
	}

	result, err := FileEntryForId(fileId, testClient)
	if err != nil {
		t.Errorf("FileEntryForId unexpectedly failed with %s", err)
	} else {
		if *result != ent {
			t.Error("Retrieved value did not match test data")
			spew.Dump(ent)
			spew.Dump(*result)
		}
	}
}

func doesListContain(haystack []FileEntry, needle FileEntry) bool {
	for _, ent := range haystack {
		if ent == needle {
			return true
		}
	}
	return false
}

func TestFilesForJobContainer(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	fileIdList := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.New(),
	}
	jobId := uuid.New()
	entries := []FileEntry{{
		Id:             fileIdList[0],
		ServerPath:     "path/to/some/file",
		JobContainerId: jobId,
		FileType:       TYPE_ORIGINAL,
		MimeType:       "video/mp4",
		Size:           123456,
	},
		{
			Id:             fileIdList[1],
			ServerPath:     "path/to/some/file.jpg",
			JobContainerId: jobId,
			FileType:       TYPE_THUMBNAIL,
			MimeType:       "image/jpeg",
			Size:           1234,
		},
		{
			Id:             fileIdList[2],
			ServerPath:     "path/to/some/file.xml",
			JobContainerId: jobId,
			FileType:       TYPE_SIDECAR,
			MimeType:       "text/xml",
			Size:           123,
		},
	}

	indexKey := fmt.Sprintf("mediaflipper:jobfile:%s", jobId)
	for _, ent := range entries {
		dbKey := fmt.Sprintf("mediaflipper:fileentry:%s", ent.Id.String())
		jsonString, _ := json.Marshal(&ent)
		setErr := s.Set(dbKey, string(jsonString))
		if setErr != nil {
			t.Errorf("TEST ERROR: could not write test record: %s", setErr)
			t.FailNow()
		}
		s.HSet(indexKey, string(ent.FileType), ent.Id.String())
	}

	results, getErr := FilesForJobContainer(jobId, testClient)
	if getErr != nil {
		t.Errorf("FilesForJobContainer failed with %s", getErr)
	} else {
		//ordering is not guaranteed
		for i, ent := range entries {
			if !doesListContain(*results, ent) {
				t.Errorf("Returned list did not contain entry %d: %s", i, spew.Sdump(ent))
			}
		}
	}
}
