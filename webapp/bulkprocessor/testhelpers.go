package bulkprocessor

import (
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"sync"
	"time"
)

/**
Mock implementation of BulkListDAO, to inject instances of BulkListMock when doing testing.
This should get eliminated by DCE (dead code elimination) during compiling/linking
*/
type BulkListDAOMock struct {
	alwaysRequestedResult bulk_models.BulkList
	singleRequestError    error
	scanResults           []*bulk_models.BulkListImpl
	scanRequestError      error
}

func (dao BulkListDAOMock) BulkListForId(bulkId uuid.UUID, client redis.Cmdable) (bulk_models.BulkList, error) {
	if dao.singleRequestError != nil {
		return nil, dao.singleRequestError
	} else {
		return dao.alwaysRequestedResult, nil
	}
}

func (dao BulkListDAOMock) ScanBulkList(start int64, stop int64, client redis.Cmdable) ([]*bulk_models.BulkListImpl, error) {
	if dao.scanRequestError != nil {
		return nil, dao.scanRequestError
	} else {
		return dao.scanResults, nil
	}
}

func (dao BulkListDAOMock) UpdateById(bulkId uuid.UUID, itemId uuid.UUID, newState bulk_models.BulkItemState, redisClient redis.Cmdable) error {
	return errors.New("mock does not implement UpdateById")
}

func (dao BulkListDAOMock) ReindexRecord(listId uuid.UUID, record bulk_models.BulkItem, oldRecord bulk_models.BulkItem, redisClient redis.Cmdable) error {
	return errors.New("mock does not implement ReindexRecord")
}

/**
Mock implementation of BulkList, to inject direclty controllable instances when doing testing
and eliminate issues that come from expecting miniredis to behave _exactly_ like real redis (e.g. sort)
*/
type BulkListMock struct {
	BulkListId     uuid.UUID
	CreationTime   time.Time
	CallCountMap   map[string]int
	callCountMutex sync.Mutex
	CallArgsMap    map[string][][]string
	allRecordsList []bulk_models.BulkItem
}

func (l *BulkListMock) DequeueContentsAsync() chan error {
	rtn := make(chan error, 1)
	rtn <- errors.New("mock does not implement dequeueContents")
	return rtn
}

func (l *BulkListMock) testNotImplementedSync() error {
	return errors.New("not implemented in test mock")
}

func (l *BulkListMock) testNotImplementedAsync() chan error {
	rtn := make(chan error)
	go func() {
		rtn <- errors.New("not implemented in test mock")
	}()
	return rtn
}

/**
increment call counter and arguments tracker
*/
func (l *BulkListMock) incrementCallCount(funcName string, args []string) {
	l.callCountMutex.Lock()
	defer l.callCountMutex.Unlock()

	curValue, exists := l.CallCountMap[funcName]
	if !exists {
		l.CallCountMap[funcName] = 1
	} else {
		l.CallCountMap[funcName] = curValue + 1
	}

	_, argsExist := l.CallArgsMap[funcName]
	if !argsExist {
		l.CallArgsMap[funcName] = make([][]string, 1)
		copy(args, l.CallArgsMap[funcName][0])
	} else {
		l.CallArgsMap[funcName] = append(l.CallArgsMap[funcName], args)
	}
}

func (l *BulkListMock) GetAllRecords(redisClient redis.Cmdable) ([]bulk_models.BulkItem, error) {
	return l.allRecordsList, nil
}

func (l *BulkListMock) GetAllRecordsAsync(redisClient redis.Cmdable) (chan bulk_models.BulkItem, chan error) {
	l.incrementCallCount("GetAllRecordsAsync", []string{})
	outputChan := make(chan bulk_models.BulkItem, 5)
	errChan := make(chan error)

	go func() {
		for _, item := range l.allRecordsList {
			outputChan <- item
		}
		outputChan <- nil
	}()
	return outputChan, errChan
}

func (l *BulkListMock) GetSpecificRecordSync(itemId uuid.UUID, redisClient redis.Cmdable) (bulk_models.BulkItem, error) {
	return nil, errors.New("mock does not implement this")
}
func (l *BulkListMock) GetSpecificRecordAsync(itemId uuid.UUID, redisClient redis.Cmdable) (chan bulk_models.BulkItem, chan error) {
	return nil, l.testNotImplementedAsync()
}

func (l *BulkListMock) FilterRecordsByState(state bulk_models.BulkItemState, redisClient redis.Cmdable) ([]bulk_models.BulkItem, error) {
	return nil, l.testNotImplementedSync()
}
func (l *BulkListMock) FilterRecordsByStateAsync(state bulk_models.BulkItemState, redisClient redis.Cmdable) (chan bulk_models.BulkItem, chan error) {
	return nil, l.testNotImplementedAsync()
}
func (l *BulkListMock) FilterRecordsByName(name string, redisClient redis.Cmdable) ([]bulk_models.BulkItem, error) {
	return nil, l.testNotImplementedSync()
}
func (l *BulkListMock) FilterRecordsByNameAsync(name string, redisClient redis.Cmdable) (chan bulk_models.BulkItem, chan error) {
	return nil, l.testNotImplementedAsync()
}
func (l *BulkListMock) FilterRecordsByNameAndStateAsync(name string, state bulk_models.BulkItemState, redisClient redis.Cmdable) (chan bulk_models.BulkItem, chan error) {
	return nil, l.testNotImplementedAsync()
}
func (l *BulkListMock) CountForState(state bulk_models.BulkItemState, redisClient redis.Cmdable) (int64, error) {
	return 0, nil
}
func (l *BulkListMock) CountForAllStates(redisClient redis.Cmdable) (map[bulk_models.BulkItemState]int64, error) {
	return nil, l.testNotImplementedSync()
}
func (l *BulkListMock) UpdateState(bulkItemId uuid.UUID, newState bulk_models.BulkItemState, redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}
func (l *BulkListMock) AddRecord(record bulk_models.BulkItem, redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}
func (l *BulkListMock) RemoveRecord(record bulk_models.BulkItem, redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}
func (l *BulkListMock) ReindexRecord(record bulk_models.BulkItem, oldRecord bulk_models.BulkItem, redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}

func (l *BulkListMock) ExistsInIndex(id uuid.UUID, redisClient redis.Cmdable) (bool, error) {
	return false, l.testNotImplementedSync()
}
func (l *BulkListMock) RebuildSortedIndex(redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}
func (l *BulkListMock) GetId() uuid.UUID {
	return l.BulkListId
}
func (l *BulkListMock) GetCreationTime() time.Time {
	return l.CreationTime
}
func (l *BulkListMock) Store(redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}
func (l *BulkListMock) Delete(redisClient redis.Cmdable) error {
	return l.testNotImplementedSync()
}

func (l *BulkListMock) SetActionRunning(actionName bulk_models.BulkListAction, redisClient redis.Cmdable) error {
	l.incrementCallCount("SetActionRunning", []string{string(actionName)})
	return nil
}
func (l *BulkListMock) ClearActionRunning(actionName bulk_models.BulkListAction, redisClient redis.Cmdable) error {
	l.incrementCallCount("ClearActionRunning", []string{string(actionName)})
	return nil
}

func (l *BulkListMock) GetActionsRunning(redisClient redis.Cmdable) ([]bulk_models.BulkListAction, error) {
	return nil, l.testNotImplementedSync()
}

func (l *BulkListMock) GetNickName() string {
	return "not implemented"
}
func (l *BulkListMock) SetNickName(newName string) {

}
func (l *BulkListMock) GetVideoTemplateId() uuid.UUID {
	return uuid.UUID{}
}
func (l *BulkListMock) SetVideoTemplateId(newId uuid.UUID) {

}
func (l *BulkListMock) GetAudioTemplateId() uuid.UUID {
	return uuid.UUID{}
}
func (l *BulkListMock) SetAudioTemplateId(newId uuid.UUID) {}
func (l *BulkListMock) GetImageTemplateId() uuid.UUID {
	return uuid.UUID{}
}
func (l *BulkListMock) SetImageTemplateId(newId uuid.UUID) {}
