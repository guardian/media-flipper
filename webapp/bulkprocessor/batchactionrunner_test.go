package bulkprocessor

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"sync"
	"testing"
	"time"
)

func TestRunAsyncActionForBatch_working(t *testing.T) {
	//RunAsyncForBatch should call the provided function once for each item belonging to the batch
	listId := uuid.MustParse("1222483E-E014-4EE5-B084-8616719C218B")

	itemIds := []uuid.UUID{
		uuid.MustParse("519C02F8-381E-4ABF-9C03-479B098055B6"),
		uuid.MustParse("5828CAB9-F0E1-4618-8412-2F2091D5B11F"),
		uuid.MustParse("093AAD85-3EE2-4FE0-A0A2-D29132A54BA0"),
		uuid.MustParse("D7FAFBFB-32BF-46BA-BB16-B4D47164508D"),
		uuid.MustParse("A7E2506C-842D-4A95-8581-9D7D21315466"),
	}

	mockRecordList := make([]BulkItem, len(itemIds))
	for i, itemId := range itemIds {
		mockRecordList[i] = &BulkItemImpl{
			Id:         itemId,
			BulkListId: listId,
			SourcePath: fmt.Sprintf("path/to/file%d", i),
			Priority:   1234,
			State:      ITEM_STATE_PENDING,
			Type:       ITEM_TYPE_IMAGE,
		}
	}

	mockedList := BulkListMock{
		BulkListId:     listId,
		CreationTime:   time.Now(),
		CallCountMap:   make(map[string]int),
		callCountMutex: sync.Mutex{},
		CallArgsMap:    make(map[string][][]string),
		allRecordsList: mockRecordList,
	}

	dao := BulkListDAOMock{alwaysRequestedResult: &mockedList}

	returnList := make([]BulkItem, 0)

	completionChan := RunAsyncActionForBatch(dao,
		listId,
		REMOVE_NONTRANSCODABLE_FILES,
		nil,
		func(itemsChan chan BulkItem, errChan chan error, outputChan chan error, list BulkList, redisClient redis.Cmdable) {
			for {
				select {
				case item := <-itemsChan:
					log.Printf("got %s", spew.Sdump(item))
					if item == nil {
						outputChan <- nil
						return
					}
					returnList = append(returnList, item)
				case err := <-errChan:
					outputChan <- err
				}
			}
		})

	returnedErr := <-completionChan

	if returnedErr != nil {
		t.Error("async callback was given an error: ", returnedErr)
	}
	if len(returnList) < 5 {
		t.Errorf("got too few items recorded, expected 5 got %d", len(returnList))
	} else {
		for i := 0; i < 5; i++ {
			if returnList[i].GetId() != itemIds[i] {
				t.Errorf("item %d had wrong id, expected %s got %s", i, itemIds[i], returnList[i].GetId())
			}
		}
		if len(returnList) > 5 {
			t.Errorf("got too many items recorded, expected 5 got %d", len(returnList))
		}
	}
}
