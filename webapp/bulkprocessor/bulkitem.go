package bulkprocessor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
)

type BulkItemState int

const (
	ITEM_STATE_PENDING BulkItemState = iota
	ITEM_STATE_ACTIVE
	ITEM_STATE_COMPLETED
	ITEM_STATE_FAILED
)

var ItemStates = []BulkItemState{
	ITEM_STATE_PENDING,
	ITEM_STATE_ACTIVE,
	ITEM_STATE_COMPLETED,
	ITEM_STATE_FAILED,
}

type BulkItem interface {
	Store(client redis.Cmdable) error
	SetState(newState BulkItemState)
	GetState() BulkItemState
	UpdateBulkItemId(newId uuid.UUID)
	GetId() uuid.UUID
	GetSourcePath() string
	GetPriority() int32
}

type BulkItemImpl struct {
	Id         uuid.UUID     `json:"id"`
	BulkListId uuid.UUID     `json:"bulkListId"`
	SourcePath string        `json:"sourcePath"`
	Priority   int32         `json:"priority"`
	State      BulkItemState `json:"state"`
}

func (i *BulkItemImpl) GetId() uuid.UUID {
	return i.Id
}

func (i *BulkItemImpl) GetSourcePath() string {
	return i.SourcePath
}

func (i *BulkItemImpl) GetPriority() int32 {
	return i.Priority
}

func NewBulkItem(filepath string, priorityOverride int32) BulkItem {
	var prio int32
	if priorityOverride > 0 {
		prio = priorityOverride
	} else {
		var char4 byte
		if len(filepath) < 4 {
			char4 = 0
		} else {
			char4 = filepath[3]
		}
		var char3 byte
		if len(filepath) < 3 {
			char3 = 0
		} else {
			char3 = filepath[2]
		}
		var char2 byte
		if len(filepath) < 2 {
			char2 = 0
		} else {
			char2 = filepath[1]
		}
		barray := []byte{filepath[0], char2, char3, char4}
		temp, _ := binary.ReadVarint(bytes.NewBuffer(barray))
		prio = int32(temp)
	}
	return &BulkItemImpl{
		Id:         uuid.New(),
		SourcePath: filepath,
		Priority:   prio,
	}
}

/**
stores the given record in the datastore.
does NOT perform indexing and should threfore be considered internal; use the methods in BulkList to store and retrive BulkItems.
takes a redis.Cmdable, which could be a pointer to a redis client or a redis Pipeliner
*/
func (i *BulkItemImpl) Store(client redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:bulkitem:%s", i.Id.String())

	content, _ := json.Marshal(i)

	_, err := client.Set(dbKey, string(content), -1).Result()
	return err
}

func (i *BulkItemImpl) SetState(newState BulkItemState) {
	i.State = newState
}

func (i *BulkItemImpl) GetState() BulkItemState {
	return i.State
}

func (i *BulkItemImpl) UpdateBulkItemId(newId uuid.UUID) {
	i.BulkListId = newId
}
