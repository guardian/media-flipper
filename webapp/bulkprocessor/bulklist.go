package bulkprocessor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/deckarep/golang-set"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"regexp"
	"strings"
	"time"
)

type BulkListAction string

const (
	REMOVE_SYSTEM_FILES          BulkListAction = "remove-system-files"
	REMOVE_NONTRANSCODABLE_FILES BulkListAction = "remove-nontranscodable"
	JOBS_QUEUEING                BulkListAction = "jobs-queueing"
)

type BulkList interface {
	GetAllRecords(redisClient redis.Cmdable) ([]BulkItem, error)
	GetAllRecordsAsync(redisClient redis.Cmdable) (chan BulkItem, chan error)
	GetSpecificRecordAsync(itemId uuid.UUID, redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByState(state BulkItemState, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByStateAsync(state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByName(name string, redisClient redis.Cmdable) ([]BulkItem, error)
	FilterRecordsByNameAsync(name string, redisClient redis.Cmdable) (chan BulkItem, chan error)
	FilterRecordsByNameAndStateAsync(name string, state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error)
	CountForState(state BulkItemState, redisClient redis.Cmdable) (int64, error)
	CountForAllStates(redisClient redis.Cmdable) (map[BulkItemState]int64, error)
	UpdateState(bulkItemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) error
	AddRecord(record BulkItem, redisClient redis.Cmdable) error
	RemoveRecord(record BulkItem, redisClient redis.Cmdable) error
	ReindexRecord(record BulkItem, oldRecord BulkItem, redisClient redis.Cmdable) error
	ExistsInIndex(id uuid.UUID, redisClient redis.Cmdable) (bool, error)
	RebuildSortedIndex(redisClient redis.Cmdable) error
	GetId() uuid.UUID
	GetCreationTime() time.Time
	Store(redisClient redis.Cmdable) error
	Delete(redisClient redis.Cmdable) error

	SetActionRunning(actionName BulkListAction, redisClient redis.Cmdable) error
	ClearActionRunning(actionName BulkListAction, redisClient redis.Cmdable) error
	GetActionsRunning(redisClient redis.Cmdable) ([]BulkListAction, error)

	GetNickName() string
	SetNickName(newName string)
	GetVideoTemplateId() uuid.UUID
	SetVideoTemplateId(newId uuid.UUID)
	GetAudioTemplateId() uuid.UUID
	SetAudioTemplateId(newId uuid.UUID)
	GetImageTemplateId() uuid.UUID
	SetImageTemplateId(newId uuid.UUID)

	DequeueContentsAsync() chan error
}

/*
proposed indexing structure:
	- mediaflipper:bulklist:%s<bulkid>:filepathindex; SET of string of form %s<filepath>:%s<idstring>
	- mediaflipper:bulklist:%s<bulkid>:filepathindexsorted; sorted version of the above
    - mediaflipper:bulklist:%s<bulkid>:state:%s<statevalue>; ORDERED SET of bulk item UUIDs sorted by BulkItem %d<priority>
	- mediaflipper:bulkitem:%s<id> ; STRING of json blob of each item
    - mediaflipper:bulklist:timeindex; ORDERED SET of all list items b
    - mediaflipper:bulklist:%s ; STRING of json blob of list metadata
*/

type BulkListImpl struct {
	BulkListId      uuid.UUID   `json:"bulkListId"`
	CreationTime    time.Time   `json:"creationTime"`
	NickName        string      `json:"nickName"`
	VideoTemplateId uuid.UUID   `json:"videoTemplateId"`
	AudioTemplateId uuid.UUID   `json:"audioTemplateId"`
	ImageTemplateId uuid.UUID   `json:"imageTemplateId"`
	BulkListDAO     BulkListDAO `json:"-"`
}

func (list *BulkListImpl) GetId() uuid.UUID {
	return list.BulkListId
}

func (list *BulkListImpl) GetCreationTime() time.Time {
	return list.CreationTime
}

func (list *BulkListImpl) GetNickName() string {
	return list.NickName
}

func (list *BulkListImpl) SetNickName(newName string) {
	list.NickName = newName
}

func (list *BulkListImpl) GetVideoTemplateId() uuid.UUID {
	return list.VideoTemplateId
}

func (list *BulkListImpl) SetVideoTemplateId(newId uuid.UUID) {
	list.VideoTemplateId = newId
}

func (list *BulkListImpl) GetAudioTemplateId() uuid.UUID {
	return list.AudioTemplateId
}

func (list *BulkListImpl) SetAudioTemplateId(newId uuid.UUID) {
	list.AudioTemplateId = newId
}

func (list *BulkListImpl) GetImageTemplateId() uuid.UUID {
	return list.ImageTemplateId
}

func (list *BulkListImpl) SetImageTemplateId(newId uuid.UUID) {
	list.ImageTemplateId = newId
}

/**
set a flag to show that the given action is running
*/
func (list *BulkListImpl) SetActionRunning(actionName BulkListAction, redisClient redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:actions", list.BulkListId)
	_, err := redisClient.SAdd(dbKey, string(actionName)).Result()
	return err
}

/**
clear the given action running flag
*/
func (list *BulkListImpl) ClearActionRunning(actionName BulkListAction, redisClient redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:actions", list.BulkListId)
	_, err := redisClient.SRem(dbKey, string(actionName)).Result()
	return err
}

/**
return a list of all running actions
*/
func (list *BulkListImpl) GetActionsRunning(redisClient redis.Cmdable) ([]BulkListAction, error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:actions", list.BulkListId)
	results, _, err := redisClient.SScan(dbKey, 0, "", 999).Result()
	if err != nil {
		return nil, err
	}
	rtn := make([]BulkListAction, len(results))
	for i, f := range results {
		rtn[i] = BulkListAction(f)
	}
	return rtn, nil
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
	if contentErr != nil && !strings.Contains(contentErr.Error(), "redis: nil") {
		log.Printf("Could not retrieve data for some of '%s': %s", strings.Join(idList, ","), contentErr)
		return contentErr
	}

	for _, r := range results {
		recordContent, _ := r.(*redis.StringCmd).Result()

		var rec *BulkItemImpl
		marshalErr := json.Unmarshal([]byte(recordContent), &rec)
		if marshalErr != nil {
			log.Printf("WARNING BatchFetchRecords - Could not unmarshal data: %s. Offending data was: %s", marshalErr, recordContent)
			continue
		}
		*outputChan <- rec
	}

	return nil
}

func (list *BulkListImpl) GetAllRecords(redisClient redis.Cmdable) ([]BulkItem, error) {
	itemsChan, errorChan := list.GetAllRecordsAsync(redisClient)

	return asyncReceiver(itemsChan, errorChan)
}

type indexMode int

const (
	INDEX_MODE_UNSORTED indexMode = iota
	INDEX_MODE_SORTED
)

/**
get a single item asynchronously. This is here to lift specific items using the same protocol as GetAllRecordsAsync for retrying individual items
in a bulk list that failed
*/
func (list *BulkListImpl) GetSpecificRecordAsync(itemId uuid.UUID, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	outputChan := make(chan BulkItem)
	errChan := make(chan error)

	go func() {
		recordKey := fmt.Sprintf("mediaflipper:bulkitem:%s", itemId.String())

		stringContent, getErr := redisClient.Get(recordKey).Result()
		if getErr != nil {
			errChan <- getErr
			return
		}

		var rec BulkItemImpl
		marshalErr := json.Unmarshal([]byte(stringContent), &rec)
		if marshalErr != nil {
			errChan <- marshalErr
			return
		}
		outputChan <- &rec
		outputChan <- nil
	}()

	return outputChan, errChan

}

func (list *BulkListImpl) GetAllRecordsAsync(redisClient redis.Cmdable) (chan BulkItem, chan error) {
	//dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:index", list.BulkListId.String())
	var pageSize int64 = 100

	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		//count, countErr := redisClient.ZCard(dbKey).Result()
		//if countErr != nil {
		//	log.Printf("ERROR: Could not receive item count for batch %s: %s", list.BulkListId, countErr)
		//	errorChan <- countErr
		//	return
		//}
		dbKey, sortErrChan := list.getSortedIndexAsync(redisClient)

		log.Printf("waiting for index sort...")
		sortErr := <-sortErrChan
		log.Printf("wait done!")
		indexMode := INDEX_MODE_SORTED
		if sortErr != nil {
			log.Printf("could not re-sort index: %s", sortErr)
			dbKey = fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", list.BulkListId.String())
			log.Printf("dbKey is now %s", dbKey)
			indexMode = INDEX_MODE_UNSORTED
		}

		var cursor uint64 = 0
		for {
			var newCursor uint64
			var keys []string
			var scanErr error

			switch indexMode {
			case INDEX_MODE_UNSORTED:
				keys, newCursor, scanErr = redisClient.SScan(dbKey, cursor, "", pageSize).Result()
				cursor = newCursor
			case INDEX_MODE_SORTED:
				keys, scanErr = redisClient.LRange(dbKey, int64(cursor), int64(cursor)+pageSize).Result()
				if len(keys) < int(pageSize) {
					newCursor = 0
				} else {
					newCursor = 1
				}
				cursor += uint64(pageSize)
			}

			if scanErr != nil {
				log.Printf("could not scan index: %s", scanErr)
				errorChan <- scanErr
				return
			}

			idList := make([]string, len(keys))
			for i, k := range keys {
				parts := strings.Split(k, "|")
				if len(parts) != 2 {
					log.Printf("ERROR: invalid key in filepath index: %s", k)
				} else {
					idList[i] = parts[1]
				}
			}

			fetchErr := list.BatchFetchRecords(idList, &outputChan, redisClient)
			if fetchErr != nil {
				errorChan <- fetchErr
				return
			}
			if newCursor == 0 {
				break
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
asynchronously reads the records in the given state and returns them via a channel
*/
func (list *BulkListImpl) FilterRecordsByStateAsync(state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	idListChan, idListErrChan := list.filterIdsByState(state, redisClient)

	go func() {
		var idList []string
		terminate := false
		for {
			select {
			case recordId := <-idListChan:
				if recordId == nil {
					terminate = true
					break
				}
				idList = append(idList, *recordId)
			case err := <-idListErrChan:
				errorChan <- err
				return
			}
			if terminate {
				break
			}
		}

		fetchErr := list.BatchFetchRecords(idList, &outputChan, redisClient)
		if fetchErr != nil {
			errorChan <- fetchErr
		} else {
			errorChan <- nil
		}
		outputChan <- nil
	}()

	return outputChan, errorChan
}

func (list *BulkListImpl) filterIdsByState(state BulkItemState, redisClient redis.Cmdable) (chan *string, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:state:%d", list.BulkListId, state)
	var pageSize int64 = 100

	outputChan := make(chan *string, 10) //set up a buffered channel
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
			for _, id := range idList {
				newValue := id
				outputChan <- &newValue
			}
		}

		outputChan <- nil //signify that we are done reading
		log.Printf("DEBUG: done fetching ids by state")
	}()

	return outputChan, errorChan
}

func (list *BulkListImpl) FilterRecordsByName(name string, redisClient redis.Cmdable) ([]BulkItem, error) {
	itemsChan, errorChan := list.FilterRecordsByNameAsync(name, redisClient)

	return asyncReceiver(itemsChan, errorChan)
}

func (list *BulkListImpl) RebuildSortedIndex(redisClient redis.Cmdable) error {
	sourceKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", list.BulkListId)
	destKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindexsorted", list.BulkListId)
	log.Printf("Rebuilding sorted filepath index for %s...", list.BulkListId)
	_, err := redisClient.SortStore(sourceKey, destKey, &redis.Sort{
		Alpha: true,
	}).Result()
	return err
}

/**
verifies that the sorted filepath index exists, and builds it if not.
returns a string of the redis key to use, and a channel. once the channel sends a value it is clear to proceed;
nil => build succeeded, error => build failed
*/
func (list *BulkListImpl) getSortedIndexAsync(redisClient redis.Cmdable) (string, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindexsorted", list.BulkListId)
	errChan := make(chan error, 1) //if we don't buffer this channel we can't send a value before returning

	i, checkErr := redisClient.Exists(dbKey).Result()
	if checkErr != nil {
		log.Printf("could not check for existence of key: %s", checkErr)
		errChan <- checkErr
		return "", errChan
	}

	if i > 0 {
		//the index already exists
		errChan <- nil
		return dbKey, errChan
	} else {
		go func() {
			buildErr := list.RebuildSortedIndex(redisClient)
			errChan <- buildErr
		}()
		return dbKey, errChan
	}
}

/**
interrogate the indices for a name that starts with the given string, retrieve the full item data and asynchronously return it via a channel.
the first channel with yield a null when the operation is completed, or the second channel will yield a single error then terminate
if the operation fails
*/
func (list *BulkListImpl) FilterRecordsByNameAsync(namePart string, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		indexMode := INDEX_MODE_SORTED
		dbKey, sortListErrChan := list.getSortedIndexAsync(redisClient)

		log.Print("waiting for index sort...")
		sortErr := <-sortListErrChan
		log.Print("done!")
		if sortErr != nil {
			log.Printf("ERROR: could not perform index sort: %s", sortErr)
			indexMode = INDEX_MODE_UNSORTED
			dbKey = fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", list.BulkListId)
		}

		idChan, idErrChan := list.fetchIdsMatchingNames(namePart, dbKey, indexMode, redisClient)

		var idList []string
		retrieveErr := func() error {
			for {
				select {
				case nextId := <-idChan:
					if nextId == nil {
						return nil
					}
					idList = append(idList, *nextId)
				case idListErr := <-idErrChan:
					return idListErr
				}
			}
		}()

		if retrieveErr != nil {
			errorChan <- retrieveErr
			return
		}

		fetchErr := list.BatchFetchRecords(idList, &outputChan, redisClient)
		if fetchErr != nil {
			errorChan <- fetchErr
			return
		}

		outputChan <- nil //signify that we are done reading
	}()

	return outputChan, errorChan
}

/*
internal method that finds the index entries matching the given querystring
*/
func (list *BulkListImpl) filterIdsByName(queryString string, mode indexMode, xtractor *regexp.Regexp, cursor uint64, dbKey string, pageSize int64, redisClient redis.Cmdable) ([]string, uint64, error) {
	for {
		var keys []string
		var scanErr error

		switch mode {
		case INDEX_MODE_UNSORTED:
			keys, cursor, scanErr = redisClient.SScan(dbKey, cursor, queryString, pageSize).Result()

			if scanErr != nil {
				return nil, 0, scanErr
			}
		case INDEX_MODE_SORTED:
			keys, scanErr = redisClient.LRange(dbKey, int64(cursor), int64(cursor)+pageSize).Result()
			cursor += uint64(len(keys))
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
		return idList, cursor, nil
	}
}

/**
internal method to fetch the uuids of items matching the given name prefix
*/
func (list *BulkListImpl) fetchIdsMatchingNames(namePart string, dbKey string, mode indexMode, redisClient redis.Cmdable) (chan *string, chan error) {
	var pageSize int64 = 100

	outputChan := make(chan *string, 10) //set up a buffered channel
	errorChan := make(chan error)

	go func() {
		xtractor := regexp.MustCompile("(?P<sourcepath>.*)\\|(?P<itemId>[\\w\\d\\-]+)")

		var cursor uint64 = 0
		var queryString string
		if strings.HasSuffix(namePart, "*") {
			queryString = namePart
		} else {
			queryString = namePart + "|*"
		}

		for {
			idList, cursor, err := list.filterIdsByName(queryString, mode, xtractor, cursor, dbKey, pageSize, redisClient)
			if err != nil {
				errorChan <- err
				return
			}
			for _, idString := range idList {
				newString := idString //it's necessary to copy the value out here.
				// idString is mutable and changes on each iteration, so taking its address
				//will point to the mutable data not the actual value that came through
				outputChan <- &newString
			}
			if cursor == 0 {
				break
			}
		}
		outputChan <- nil
	}()
	return outputChan, errorChan
}

func (list *BulkListImpl) FilterRecordsByNameAndStateAsync(namePart string, state BulkItemState, redisClient redis.Cmdable) (chan BulkItem, chan error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", list.BulkListId)
	//var pageSize int64 = 100

	outputChan := make(chan BulkItem, 10) //set up a buffered channel
	errorChan := make(chan error)

	idsMatchingNameChan, nameMatchErrChan := list.fetchIdsMatchingNames(namePart, dbKey, INDEX_MODE_UNSORTED, redisClient)
	idsMatchingStateChan, stateMatchErrChan := list.filterIdsByState(state, redisClient)

	go func() {
		nameMatchesSet := mapset.NewThreadUnsafeSet()
		stateMatchesSet := mapset.NewThreadUnsafeSet()
		terminate := []bool{false, false}
		for {
			select {
			case idMatchingName := <-idsMatchingNameChan:
				if idMatchingName == nil {
					terminate[0] = true
				} else {
					nameMatchesSet.Add(*idMatchingName)
				}
			case nameMatchErr := <-nameMatchErrChan:
				errorChan <- nameMatchErr
				terminate[0] = true
			case idMatchingState := <-idsMatchingStateChan:
				if idMatchingState == nil {
					terminate[1] = true
				} else {
					stateMatchesSet.Add(*idMatchingState)
				}
			case idMatchErr := <-stateMatchErrChan:
				errorChan <- idMatchErr
				terminate[1] = true
			}
			if terminate[0] && terminate[1] {
				break
			}
		}

		matches := nameMatchesSet.Intersect(stateMatchesSet)

		idList := make([]string, matches.Cardinality())
		i := 0
		for value := range matches.Iter() {
			if matchingId, isOk := value.(string); isOk {
				idList[i] = matchingId
				i += 1
			}
		}

		err := list.BatchFetchRecords(idList, &outputChan, redisClient)
		if err != nil {
			errorChan <- err
			return
		}
		outputChan <- nil
	}()

	return outputChan, errorChan
}

/**
updates the state of a given item (by id) and stores/re-indexes it as necessary. Returns a pointer to the updated item.
*/
func (list *BulkListImpl) UpdateState(bulkItemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) error {
	return list.BulkListDAO.UpdateById(list.GetId(), bulkItemId, newState, redisClient)
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
	//addToGlobalIndex(record, baseKey, pipe)

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
	pipe.Del(fmt.Sprintf("mediaflipper:bulkitem:%s", record.GetId()))

	_, execErr := pipe.Exec()
	if execErr != nil {
		log.Printf("Could not execute pipelined removal: %s", execErr)
		return execErr
	} else {
		return nil
	}
}

/**
update any indices containing this record
*/
func (list *BulkListImpl) ReindexRecord(record BulkItem, oldRecord BulkItem, redisClient redis.Cmdable) error {
	return list.BulkListDAO.ReindexRecord(list.GetId(), record, oldRecord, redisClient)
}

func (list *BulkListImpl) ExistsInIndex(id uuid.UUID, redisClient redis.Cmdable) (bool, error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:index", list.BulkListId)
	rank, err := redisClient.ZRank(dbKey, id.String()).Result()
	if err != nil {
		if strings.Contains(err.Error(), "redis: nil") {
			return false, nil
		} else {
			return false, err
		}
	}
	return rank > 0, nil
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

func (list *BulkListImpl) Store(redisClient redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s", list.BulkListId)

	content, marshalErr := json.Marshal(list)
	if marshalErr != nil {
		return marshalErr
	}

	var pipe redis.Pipeliner
	if castPipeline, isAlreadyPipeline := redisClient.(redis.Pipeliner); isAlreadyPipeline {
		pipe = castPipeline
	} else {
		pipe = redisClient.Pipeline()
		defer pipe.Close()
	}

	pipe.Set(dbKey, string(content), -1)
	pipe.ZAdd("mediaflipper:bulklist:timeindex", &redis.Z{
		Score:  float64(list.CreationTime.Unix()),
		Member: list.BulkListId.String(),
	})

	if _, isAlreadyPipeline := redisClient.(redis.Pipeliner); !isAlreadyPipeline {
		_, putErr := pipe.Exec()
		if putErr != nil {
			return putErr
		}
	}
	return nil
}

/**
delete the given list and all its associated indices
*/
func (list *BulkListImpl) Delete(redisClient redis.Cmdable) error {
	pipe := redisClient.Pipeline()
	defer pipe.Close()
	baseKey := fmt.Sprintf("mediaflipper:bulklist:%s", list.BulkListId)
	//delete list-specific indices
	for _, s := range ItemStates {
		dbKey := fmt.Sprintf("%s:state:%d", baseKey, s)
		pipe.Del(dbKey)
	}
	pipe.Del(baseKey + ":filepathindex")
	pipe.Del(baseKey)
	pipe.ZRem("mediaflipper:bulklist:timeindex", list.BulkListId.String())
	_, err := pipe.Exec()
	return err
}

//create an interface to hold the "global" functions so we can easily stub them in testing
type BulkListDAO interface {
	BulkListForId(bulkId uuid.UUID, client redis.Cmdable) (BulkList, error)
	ScanBulkList(start int64, stop int64, client redis.Cmdable) ([]*BulkListImpl, error)
	UpdateById(bulkId uuid.UUID, itemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) error
	ReindexRecord(listId uuid.UUID, record BulkItem, oldRecord BulkItem, redisClient redis.Cmdable) error
}

type BulkListDAOImpl struct{}

func (dao BulkListDAOImpl) BulkListForId(bulkId uuid.UUID, client redis.Cmdable) (BulkList, error) {
	dbKey := fmt.Sprintf("mediaflipper:bulklist:%s", bulkId)

	content, getErr := client.Get(dbKey).Result()
	if getErr != nil {
		return nil, getErr
	}

	var rtn BulkListImpl
	unMarshalErr := json.Unmarshal([]byte(content), &rtn)
	if unMarshalErr != nil {
		return nil, unMarshalErr
	}
	rtn.BulkListDAO = BulkListDAOImpl{}
	return &rtn, nil
}

func (dao BulkListDAOImpl) ScanBulkList(start int64, stop int64, client redis.Cmdable) ([]*BulkListImpl, error) {
	return ScanBulkList(start, stop, client)
}

func (dao BulkListDAOImpl) UpdateById(bulkId uuid.UUID, itemId uuid.UUID, newState BulkItemState, redisClient redis.Cmdable) error {
	if _, isPipeline := redisClient.(redis.Pipeliner); isPipeline {
		return errors.New("sorry, UpdateById does not support being pipelined")
	}
	oldItem, getErr := dao.RecordForId(itemId, redisClient)
	if getErr != nil {
		return getErr
	}

	updatedItem := oldItem.CopyWithNewState(newState)
	pipe := redisClient.Pipeline()
	defer pipe.Close()

	dao.ReindexRecord(bulkId, updatedItem, oldItem, pipe)
	updatedItem.Store(pipe)

	_, setErrors := pipe.Exec()
	if setErrors != nil {
		return setErrors
	}
	return nil
}

func (dao BulkListDAOImpl) RecordForId(bulkItemId uuid.UUID, redisClient redis.Cmdable) (BulkItem, error) {
	dbKey := fmt.Sprintf("mediaflipper:bulkitem:%s", bulkItemId)
	content, err := redisClient.Get(dbKey).Result()
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, errors.New("no such record existed")
	}
	var rec BulkItemImpl
	marshalErr := json.Unmarshal([]byte(content), &rec)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return &rec, nil
}

func (dao BulkListDAOImpl) ReindexRecord(listId uuid.UUID, record BulkItem, oldRecord BulkItem, redisClient redis.Cmdable) error {
	baseKey := fmt.Sprintf("mediaflipper:bulklist:%s", listId)
	//log.Printf("DEBUG: ReindexRecord old record is %s", spew.Sdump(oldRecord))
	removeFromStateIndex(oldRecord, baseKey, redisClient)
	//log.Printf("DEBUG: ReindexRecord new record is %s", spew.Sdump(record))
	addToStateIndex(record, baseKey, redisClient)
	return nil
}

/**
load up the given bulk list
*/
func BulkListForId(bulkId uuid.UUID, client redis.Cmdable) (BulkList, error) {
	dao := BulkListDAOImpl{}
	return dao.BulkListForId(bulkId, client)
}

func ScanBulkList(start int64, stop int64, client redis.Cmdable) ([]*BulkListImpl, error) {
	idList, listErr := client.ZRange("mediaflipper:bulklist:timeindex", start, stop).Result()
	if listErr != nil {
		log.Print("could not list out timeindex: ", listErr)
		return nil, listErr
	}

	pipe := client.Pipeline()
	defer pipe.Close()

	for _, listId := range idList {
		dataKey := fmt.Sprintf("mediaflipper:bulklist:%s", listId)
		pipe.Get(dataKey)
	}

	getResults, getErr := pipe.Exec()
	if getErr != nil {
		log.Printf("could not retrieve items from datastore: %s. Item list was: %s", getErr, spew.Sdump(idList))
		return nil, getErr
	}

	results := make([]*BulkListImpl, len(getResults))
	for i, getResult := range getResults {
		r := getResult.(*redis.StringCmd)
		content, _ := r.Result()
		marshalErr := json.Unmarshal([]byte(content), &results[i])
		if marshalErr != nil {
			log.Printf("could not unmarshal data from store: %s. Offending data was %s", marshalErr, content)
			return nil, marshalErr
		}
	}

	return results, nil
}

func (l *BulkListImpl) DequeueContentsAsync() chan error {
	rtnChan := make(chan error, 1)
	rtnChan <- errors.New("not implemented yet")
	return rtnChan
}
