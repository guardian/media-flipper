package bulkprocessor

import (
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"testing"
	"time"
)

func PrepareTestData(client redis.Cmdable) BulkList {
	bulkListId := uuid.MustParse("C15507F1-1920-4C3E-861C-E9914DDEC49D")
	testRecords := []*BulkItemImpl{
		{
			uuid.MustParse("68823F20-1579-4CEF-A080-8B1942EF538A"),
			bulkListId,
			"path/to/file1",
			1,
			ITEM_STATE_COMPLETED,
		},
		{
			uuid.MustParse("AFDB2DD8-6B5F-4DEB-88A7-CBC2CD545DA6"),
			bulkListId,
			"path/to/file2",
			1,
			ITEM_STATE_ACTIVE,
		},
		{
			uuid.MustParse("599B1967-8E69-4A7B-B0E3-710053EFF5C4"),
			bulkListId,
			"path/to/file3",
			1,
			ITEM_STATE_ACTIVE,
		},
		{
			uuid.MustParse("D7285685-03D8-49CD-A4BB-924F326497DD"),
			bulkListId,
			"path/to/file4",
			1,
			ITEM_STATE_PENDING,
		},
	}

	serializedRecords := make([]string, len(testRecords))
	for i, rec := range testRecords {
		bytes, _ := json.Marshal(rec)
		serializedRecords[i] = string(bytes)
	}
	//write records
	for i, rec := range serializedRecords {
		client.Set(fmt.Sprintf("mediaflipper:bulkitem:%s", testRecords[i].Id), rec, -1)
	}

	//write state index
	for _, rec := range testRecords {
		dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:state:%d", bulkListId.String(), rec.State)
		client.ZAdd(dbKey, &redis.Z{
			Score:  float64(rec.Priority),
			Member: rec.Id.String(),
		})
	}

	//write filename index
	for _, rec := range testRecords {
		dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:filepathindex", bulkListId.String())
		dataKey := fmt.Sprintf("%s|%s", rec.SourcePath, bulkListId.String())
		client.SAdd(dbKey, dataKey)
	}

	return &BulkListImpl{
		BulkListId:   bulkListId,
		CreationTime: time.Now(),
	}
}

func TestBulkListImpl_FilterRecordsByState(t *testing.T) {
	/* prepopulate data */
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	testList := PrepareTestData(testClient)

	pendingRecords, pendingRecordsErr := testList.FilterRecordsByState(ITEM_STATE_PENDING, testClient)
	if pendingRecordsErr != nil {
		t.Error("FilterRecordsByState failed: ", pendingRecordsErr)
	} else {
		if len(pendingRecords) != 1 {
			t.Errorf("ITEM_STATE_PENDING returned wrong number of records, expected 1 got %d", len(pendingRecords))
		}
	}

	activeRecords, activeRecordsErr := testList.FilterRecordsByState(ITEM_STATE_ACTIVE, testClient)
	if activeRecordsErr != nil {
		t.Error("FilterRecordsByState failed: ", activeRecordsErr)
	} else {
		if len(activeRecords) != 2 {
			t.Errorf("ITEM_STATE_ACTIVE returned wrong number of records, expected 2 got %d", len(activeRecords))
		}
		spew.Dump(activeRecords)
	}

	completedRecords, completeRecordsErr := testList.FilterRecordsByState(ITEM_STATE_COMPLETED, testClient)
	if completeRecordsErr != nil {
		t.Error("FilterRecordsByState failed: ", completeRecordsErr)
	} else {
		if len(completedRecords) != 1 {
			t.Errorf("ITEM_STATE_COMPLETED returned wrong number of records, expected 1 got %d", len(activeRecords))
		}
		spew.Dump(completedRecords)
	}

	failedRecords, failedRecordsErr := testList.FilterRecordsByState(ITEM_STATE_FAILED, testClient)
	if failedRecordsErr != nil {
		t.Error("FilterREcordsByState failed: ", failedRecordsErr)
	} else {
		if len(failedRecords) != 0 {
			t.Errorf("ITEM_STATE_FAILED returned wrong number of records, expected 1 got %d", len(failedRecords))
		}
		spew.Dump(failedRecords)
	}

}
