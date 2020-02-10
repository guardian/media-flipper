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
			2,
			ITEM_STATE_ACTIVE,
		},
		{
			uuid.MustParse("599B1967-8E69-4A7B-B0E3-710053EFF5C4"),
			bulkListId,
			"path/to/file3",
			3,
			ITEM_STATE_ACTIVE,
		},
		{
			uuid.MustParse("D7285685-03D8-49CD-A4BB-924F326497DD"),
			bulkListId,
			"anotherpath/to/file4",
			4,
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
		dataKey := fmt.Sprintf("%s|%s", rec.SourcePath, rec.Id.String())
		client.SAdd(dbKey, dataKey)
	}

	//write sort index
	for _, rec := range testRecords {
		dbKey := fmt.Sprintf("mediaflipper:bulklist:%s:index", bulkListId.String())
		client.ZAdd(dbKey, &redis.Z{float64(rec.Priority), rec.Id.String()})
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

func TestBulkListImpl_FilterRecordsByName(t *testing.T) {
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

	prefixList, prefixListErr := testList.FilterRecordsByName("path/to*", testClient)
	if prefixListErr != nil {
		t.Error("FilterRecordsByName failed for prefix: ", prefixListErr)
	} else {
		if len(prefixList) != 3 {
			t.Errorf("Incorrect result count returned for base prefix test, expected 3 got %d", len(prefixList))
		} else {
			realContent0 := prefixList[0].(*BulkItemImpl)
			realContent1 := prefixList[1].(*BulkItemImpl)
			realContent2 := prefixList[2].(*BulkItemImpl)
			if realContent0.SourcePath != "path/to/file1" {
				t.Errorf("Got incorrect path for first result, expected path/to/file1 got %s", realContent0.SourcePath)
			}
			if realContent1.SourcePath != "path/to/file2" {
				t.Errorf("Got incorrect path for first result, expected path/to/file2 got %s", realContent1.SourcePath)
			}
			if realContent2.SourcePath != "path/to/file3" {
				t.Errorf("Got incorrect path for first result, expected path/to/file3 got %s", realContent2.SourcePath)
			}
		}
	}

	exactMatchList, exactMatchErr := testList.FilterRecordsByName("path/to/file1", testClient)
	if exactMatchErr != nil {
		t.Error("FilterRecordsByName failed for exact: ", exactMatchErr)
	} else {
		if len(exactMatchList) != 1 {
			t.Errorf("Incorrect result count for exact match test, expected 1 got %d", len(exactMatchList))
		} else {
			realContent := prefixList[0].(*BulkItemImpl)
			if realContent.Id != uuid.MustParse("68823F20-1579-4CEF-A080-8B1942EF538A") {
				t.Errorf("Incorrect item returned for exact match test, expected 68823F20-1579-4CEF-A080-8B1942EF538A got %s", realContent.Id)
			}
		}
	}

	noMatchList, noMatchErr := testList.FilterRecordsByName("sdfkjhsdfjkhsfjhksfd", testClient)
	if noMatchErr != nil {
		t.Error("FilterRecordsByName failed for no matches: ", noMatchErr)
	} else {
		if len(noMatchList) != 0 {
			t.Errorf("Incorrect result countm for exact match test, expected 0 got %d", len(noMatchList))
		}
	}
}

func TestBulkListImpl_GetAllRecords(t *testing.T) {
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

	allRecordsList, allRecordsErr := testList.GetAllRecords(testClient)
	if allRecordsErr != nil {
		t.Error("GetAllRecords unexpectedly failed: ", allRecordsErr)
	} else {
		if len(allRecordsList) != 4 {
			t.Errorf("Got incorrect number of results, expected 4 got %d", len(allRecordsList))
		}
		realData := make([]*BulkItemImpl, len(allRecordsList))
		for i, rec := range allRecordsList {
			realData[i] = rec.(*BulkItemImpl)
		}
		if realData[0].SourcePath != "path/to/file1" {
			t.Errorf("Got unexpected source path for item 0, expected path/to/file1 got %s", realData[0].SourcePath)
		}
		if realData[1].SourcePath != "path/to/file2" {
			t.Errorf("Got unexpected source path for item 1, expected path/to/file2 got %s", realData[1].SourcePath)
		}
		if realData[2].SourcePath != "path/to/file3" {
			t.Errorf("Got unexpected source path for item 2, expected path/to/file3 got %s", realData[2].SourcePath)
		}
		if realData[3].SourcePath != "anotherpath/to/file4" {
			t.Errorf("Got unexpected source path for item 3, expected anotherpath/to/file4 got %s", realData[3].SourcePath)
		}
	}
}
