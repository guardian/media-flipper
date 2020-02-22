package bulkprocessor

import (
	"github.com/google/uuid"
	"testing"
)

func TestNewBulkItem(t *testing.T) {
	//NewBulkItem should return a BulkItem instance with the uuid initialised and an auto-calculated priority
	testItem := NewBulkItem("path/to/somefile", -1)
	realItem := testItem.(*BulkItemImpl)

	blankId := uuid.UUID{}
	if realItem.Id == blankId {
		t.Error("created item did not have a valid uuid")
	}
	if realItem.BulkListId != blankId {
		t.Error("newly created item list id was not blank")
	}
	if realItem.SourcePath != "path/to/somefile" {
		t.Errorf("newly created item path was wrong, expected path/to/somefile, got %s", realItem.SourcePath)
	}
	if realItem.State != ITEM_STATE_NOT_QUEUED {
		t.Errorf("newly created item state was wrong, expected ITEM_STATE_NOT_QUEUED got %d", realItem.State)
	}
	if realItem.Priority != 1885434984 {
		t.Errorf("got unexpected value for item priority, expected 1885434984 got %d", realItem.Priority)
	}

	//NewBulkItem should respect priority override
	testItem2 := NewBulkItem("path/to/somefile", 243)
	realItem2 := testItem2.(*BulkItemImpl)

	if realItem2.Id == blankId {
		t.Error("created item did not have a valid uuid")
	}
	if realItem2.BulkListId != blankId {
		t.Error("newly created item list id was not blank")
	}
	if realItem2.SourcePath != "path/to/somefile" {
		t.Errorf("newly created item path was wrong, expected path/to/somefile, got %s", realItem.SourcePath)
	}
	if realItem2.State != ITEM_STATE_NOT_QUEUED {
		t.Errorf("newly created item state was wrong, expected ITEM_STATE_NOT_QUEUED got %d", realItem.State)
	}
	if realItem2.Priority != 243 {
		t.Errorf("got unexpected value for item priority, expected 1885434984 got %d", realItem.Priority)
	}
}
