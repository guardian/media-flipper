package bulkprocessor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"regexp"
	"strings"
	"time"
)

type BulkList interface {
	GetAllRecords(redisClient redis.Cmdable) ([]BulkItem, error)
	GetAllRecordsAsync(redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByState(state BulkItemState, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByStateAsync(state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByName(name string, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByNameAsync(name string, redisClient redis.Cmdable) (chan BulkItem, chan error)
	CountForState(state BulkItemState, redisClient redis.Cmdable) (int64, error)
	CountForAllStates(redisClient redis.Cmdable) (map[BulkItemState]int64, error)
	UpdateState(bulkItemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) (*BulkItem, error)
	AddRecord(record BulkItem, redisClient redis.Cmdable) error
	RemoveRecord(record BulkItem, redisClient redis.Cmdable) error
	GetId() uuid.UUID
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

func (list *BulkListImpl) GetId() uuid.UUID {
	return list.BulkListId
}
func (list *BulkListImpl) GetAllRecords(redisClient redis.Cmdable) ([]BulkItem, error) {
	itemsChan, errorChan := list.GetAllRecordsAsync(redisClient)

	return asyncReceiver(itemsChan, errorChan)
}

func (list *BulkListImpl) GetAllRecordsAsync(redisClient redis.Cmdable) (chan BulkItem, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:index", list.BulkListId.String())
	var pageSize int64 = 100

	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		count, countErr := redisClient.ZCard(dbKey).Result()
		if countErr != nil {
			log.Printf("ERROR: Could not receive item count for batch %s: %s", list.BulkListId, countErr)
			errorChan <- countErr
			return
		}
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
		log.Printf("Could not retrieve data for some of '%s': %s", strings.Join(idList, ","), contentErr)
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
		xtractor := regexp.MustCompile("(?P<sourcepath>.*)\\|(?P<itemId>[\\w\\d\\-]+)")

		var queryString string
		if strings.HasSuffix(namePart, "*") {
			queryString = namePart
		} else {
			queryString = namePart + "|*"
		}

		var cursor uint64 = 0
		for {
			var keys []string
			var scanErr error

			keys, cursor, scanErr = redisClient.SScan(dbKey, cursor, queryString, pageSize).Result()

			if scanErr != nil {
				errorChan <- scanErr
				return
			}

			idList := make([]string, len(keys))
			for i, key := range keys {
				xtracted := xtractor.FindAllStringSubmatch(key, -1)
				if xtracted == nil {
					log.Printf("WARNING: Invalid data in filepath index: %s", key)
				} else {
					//we are only interested in validating that the data parses as a uuid, as we'd only have to convert it straight
					//back again afterwards
					_, uuidErr := uuid.Parse(xtracted[0][2])
					if uuidErr != nil {
						log.Printf("WARNING: could not parse uuid: %s", uuidErr)
					} else {
						idList[i] = xtracted[0][2]
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

func addToFilePathIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + ":filepathindex"
	dbVal := fmt.Sprintf("%s|%s", record.GetSourcePath(), record.GetId().String())
	redisClient.SAdd(dbKey, dbVal)
}

func removeFromFilePathIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + ":filepathindex"
	dbVal := fmt.Sprintf("%s|%s", record.GetSourcePath(), record.GetId().String())
	redisClient.SRem(dbKey, dbVal)
}

func addToStateIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + fmt.Sprintf(":state:%d", record.GetState())
	redisClient.ZAdd(dbKey, &redis.Z{
		float64(record.GetPriority()),
		record.GetId().String(),
	})
}

func removeFromStateIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + fmt.Sprintf(":state:%d", record.GetState())
	redisClient.ZRem(dbKey, record.GetId().String())
}

func addToGlobalIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + ":index"
	redisClient.ZAdd(dbKey, &redis.Z{
		float64(record.GetPriority()),
		record.GetId().String(),
	})
}

func removeFromGlobalIndex(record BulkItem, baseKey string, redisClient redis.Cmdable) {
	dbKey := baseKey + ":index"
	redisClient.ZRem(dbKey, record.GetId().String())
}

/**
add the given record to the bulk list. the record is modified to give the id if this list and both saved and indexed
*/
func (list *BulkListImpl) AddRecord(record BulkItem, redisClient redis.Cmdable) error {
	record.UpdateBulkItemId(list.BulkListId)
	pipe := redisClient.Pipeline()
	defer pipe.Close()

	baseKey := fmt.Sprintf("mediaflipper:bulklist:%s", list.BulkListId)
	addToFilePathIndex(record, baseKey, pipe)
	addToStateIndex(record, baseKey, pipe)
	addToGlobalIndex(record, baseKey, pipe)

	record.Store(pipe) //no point looking for error here as it is only executed at the next step

	_, execErr := pipe.Exec()
	if execErr != nil {
		log.Printf("Could not complete add record: %s", execErr)
		return execErr
	} else {
		return nil
	}
}

/**
remove the given record from the bulk list, datastore and indices
*/
func (list *BulkListImpl) RemoveRecord(record BulkItem, redisClient redis.Cmdable) error {
	if record.GetBulkId() != list.BulkListId {
		return errors.New(fmt.Sprintf("the record %s is not associated with bulk list %s. Association is %s.", record.GetId(), list.BulkListId, record.GetBulkId()))
	}

	pipe := redisClient.Pipeline()
	defer pipe.Close()

	baseKey := fmt.Sprintf("mediaflipper:bulklist:%s", list.BulkListId)
	removeFromFilePathIndex(record, baseKey, pipe)
	removeFromStateIndex(record, baseKey, pipe)
	removeFromGlobalIndex(record, baseKey, pipe)
	pipe.Del(fmt.Sprintf("mediaflipper:bulkitem:%s", record.GetId()))

	_, execErr := pipe.Exec()
	if execErr != nil {
		log.Printf("Could not execute pipelined removal: %s", execErr)
		return execErr
	} else {
		return nil
	}
}

func (list *BulkListImpl) CountForState(state BulkItemState, redisClient redis.Cmdable) (int64, error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:state:%d", list.BulkListId, state)
	return redisClient.ZCard(dbKey).Result()
}

func (list *BulkListImpl) CountForAllStates(redisClient redis.Cmdable) (map[BulkItemState]int64, error) {
	rtn := make(map[BulkItemState]int64, len(ItemStates))
	pipe := redisClient.Pipeline()
	defer pipe.Close()
	for _, s := range ItemStates {
		dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:state:%d", list.BulkListId, s)
		pipe.ZCard(dbKey)
	}

	results, execErr := pipe.Exec()
	if execErr != nil {
		log.Printf("ERROR: Could not exec redis query: %s", execErr)
		return nil, execErr
	}

	for i, result := range results {
		realResult := result.(*redis.IntCmd)
		state := BulkItemState(i)
		rtn[state] = realResult.Val()
	}
	return rtn, nil
}
