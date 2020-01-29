package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"testing"
)

func TestNewFileEntry(t *testing.T) {
	fakeId := uuid.New()
	existingFile, err := NewFileEntry("fileentry.go", fakeId, TYPE_ORIGINAL)

	if err != nil {
		t.Error("File entry for existing file failed: ", err)
	} else {
		spew.Dump(existingFile)
	}
}
