package bulkprocessor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"regexp"
	"time"
)

type BulkList interface {
	GetAllRecords(redisClient redis.Cmdable) (*[]BulkItem, error)
	GetAllRecordsAsync(redisClient redis.Cmdable) (chan *BulkItem, chan error)
	FilterRecordsByState(state BulkItemState, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByStateAsync(state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByName(name string, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByNameAsync(name string, redisClient redis.Cmdable) (chan BulkItem, chan error)
	UpdateState(bulkItemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) (*BulkItem, error)
	AddRecord(record *BulkItem, redisClient redis.Cmdable) error
	RemoveRecord(record *BulkItem, redisClient redis.Cmdable) error
}

/*
proposed indexing structure:
	- mediaflipper:bulklist:%s<bulkid>:filepathindex; SET of string of form %s<filepath>:%s<idstring>
    - mediaflipper:bulklist:%s<bulkid>:state:%s<statevalue>; ORDERED SET of bulk item UUIDs sorted by BulkItem %d<priority>
    - mediaflipper:bulklist:%s<bulkid>:index; ORDERED SET string of bulk item UUIDs sorted by BulkItem %d<priority>
	- mediaflipper:bulkitem:%s<id> ; STRING of json blob of each item
    - mediaflipper:bulklist:timeindex; ORDERED SET of all list items b
    - mediaflipper:bulklist:%s ; STRING of json blob of list metadata
*/

type BulkListImpl struct {
	BulkListId   uuid.UUID
	CreationTime time.Time
}

func (list *BulkListImpl) GetAllRecords(redisClient redis.Cmdable) (*[]BulkItem, error) {
	return nil, errors.New("not implemented")
}

func (list *BulkListImpl) GetAllRecordsAsync(redisClient redis.Cmdable) (chan *BulkItem, chan error) {
	outputChan := make(chan *BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	errorChan <- errors.New("not implemented")
	return outputChan, errorChan
}

/**
synchronous version of FilterRecordsByStateAsync that sets up a return loop for the async function and marshals the
stream into an array of record pointers
*/
func (list *BulkListImpl) FilterRecordsByState(state BulkItemState, redisClient redis.Cmdable) ([]BulkItem, error) {
	itemsChan, errorChan := list.FilterRecordsByStateAsync(state, redisClient)

	return asyncReceiver(itemsChan, errorChan)
}

/**
generic receiver to marshal a stream of BulkItem and error into a stream of BulkItem or a single error
(terminates on first error receive, or on a nil receive from itemsChan)
*/
func asyncReceiver(itemsChan chan BulkItem, errorChan chan error) ([]BulkItem, error) {
	var rtn []BulkItem
	for {
		select {
		case newItem := <-itemsChan:
			if newItem == nil {
				log.Printf("Received all items")
				return rtn, nil
			} else {
				rtn = append(rtn, newItem)
			}
		case scanErr := <-errorChan:
			log.Printf("Receved async error: %s", scanErr)
			return nil, scanErr
		}
	}
}

/**
generic function to do a pipelined retrieve of BulkItem records, given an array of uuids to lift
each 'hit' is pushed sequentially to `outputChan`.
*/
func (list *BulkListImpl) BatchFetchRecords(idList []string, outputChan *chan BulkItem, redisClient redis.Cmdable) error {
	pipe := redisClient.Pipeline()

	for _, itemId := range idList {
		recordKey := fmt.Sprintf("mediaflipper:bulkitem:%s", itemId)
		pipe.Get(recordKey)
	}

	results, contentErr := pipe.Exec()
	defer pipe.Close()
	if contentErr != nil {
		log.Printf("Could not retrieve data for %s", contentErr)
		return contentErr
	}

	for _, r := range results {
		recordContent, _ := r.(*redis.StringCmd).Result()

		var rec *BulkItemImpl
		marshalErr := json.Unmarshal([]byte(recordContent), &rec)
		if marshalErr != nil {
			log.Printf("Could not unmarshal data: %s. Offending data was: %s", marshalErr, recordContent)
			continue
		}
		*outputChan <- rec
	}

	return nil
}

/**
asynchronously reads the records in the given state and returns them via a channel
*/
func (list *BulkListImpl) FilterRecordsByStateAsync(state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:state:%d", list.BulkListId, state)
	var pageSize int64 = 100

	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		count, countErr := redisClient.ZCard(dbKey).Result()
		if countErr != nil {
			log.Printf("ERROR: Could not retrieve item count for %s: %s", dbKey, countErr)
			errorChan <- countErr
			return
		}

		log.Printf("DEBUG: Got %d records for %s", count, dbKey)

		var i int64
		for i = 0; i < count; i += pageSize {
			idList, idListErr := redisClient.ZRange(dbKey, i, i+pageSize).Result()
			if idListErr != nil {
				errorChan <- idListErr
				return
			}

			fetchErr := list.BatchFetchRecords(idList, &outputChan, redisClient)
			if fetchErr != nil {
				errorChan <- fetchErr
				return
			}
		}

		outputChan <- nil //signify that we are done reading
	}()

	return outputChan, errorChan
}

func (list *BulkListImpl) FilterRecordsByName(name string, redisClient redis.Cmdable) ([]BulkItem, error) {
	itemsChan, errorChan := list.FilterRecordsByNameAsync(name, redisClient)

	return asyncReceiver(itemsChan, errorChan)
}

/**
interrogate the indices for a name that starts with the given string, retrieve the full item data and asynchronously return it via a channel.
the first channel with yield a null when the operation is completed, or the second channel will yield a single error then terminate
if the operation fails
*/
func (list *BulkListImpl) FilterRecordsByNameAsync(namePart string, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", list.BulkListId)
	var pageSize int64 = 100

	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		xtractor := regexp.MustCompile("(?P<sourcepath>.*)|(?P<itemId>[\\w\\d\\-]+)")

		var cursor uint64 = 0
		for {
			var keys []string
			var scanErr error

			keys, cursor, scanErr = redisClient.SScan(dbKey, cursor, namePart, pageSize).Result()

			if scanErr != nil {
				errorChan <- scanErr
				return
			}

			idList := make([]string, len(keys))
			for i, key := range keys {
				xtracted := xtractor.FindStringSubmatch(key)
				if xtracted == nil {
					log.Printf("WARNING: Invalid data in filepath index: %s", key)
				} else {
					//we are only interested in validating that the data parses as a uuid, as we'd only have to convert it straight
					//back again afterwards
					_, uuidErr := uuid.Parse(xtracted[1])
					if uuidErr != nil {
						log.Printf("WARNING: could not parse uuid: %s", uuidErr)
					} else {
						idList[i] = xtracted[1]
					}
				}
			}

			fetchErr := list.BatchFetchRecords(idList, &outputChan, redisClient)
			if fetchErr != nil {
				errorChan <- fetchErr
				return
			}
			if cursor == 0 {
				break
			}
		}

		outputChan <- nil //signify that we are done reading
	}()

	return outputChan, errorChan
}

/**
updates the state of a given item (by id) and stores/re-indexes it as necessary. Returns a pointer to the updated item.
*/
func (list *BulkListImpl) UpdateState(bulkItemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) (*BulkItem, error) {
	return nil, errors.New("not implemented")
}

/**
add the given record to the bulk list. the record is modified to give the id if this list and both saved and indexed
*/
func (list *BulkListImpl) AddRecord(record *BulkItem, redisClient redis.Cmdable) error {
	return errors.New("not implemented")
}

/**
remove the given record from the bulk list, datastore and indices
*/
func (list *BulkListImpl) RemoveRecord(record *BulkItem, redisClient redis.Cmdable) error {
	return errors.New("not implemented")
}
