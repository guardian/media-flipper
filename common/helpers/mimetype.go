package helpers

import (
	"mime"
	"regexp"
	"strings"
	"sync"
)

type BulkItemType string

const (
	ITEM_TYPE_VIDEO BulkItemType = "video"
	ITEM_TYPE_AUDIO BulkItemType = "audio"
	ITEM_TYPE_IMAGE BulkItemType = "image"
	ITEM_TYPE_OTHER BulkItemType = "other"
)

var FileExtensionExtractor = regexp.MustCompile("(\\.[^\\.]+)$")
var once sync.Once

func ItemTypeForFilepath(filepath string) BulkItemType {
	once.Do(func() {
		mime.AddExtensionType(".mxf", "video/x-material-exchange-format")
		mime.AddExtensionType(".mts", "video/x-mpeg-transport-stream")
	})

	var itemType BulkItemType
	matches := FileExtensionExtractor.FindStringSubmatch(filepath)
	if matches == nil {
		itemType = ITEM_TYPE_OTHER
	} else {
		mimeType := mime.TypeByExtension(matches[1])
		if strings.HasPrefix(mimeType, "video/") {
			itemType = ITEM_TYPE_VIDEO
		} else if strings.HasPrefix(mimeType, "audio/") {
			itemType = ITEM_TYPE_AUDIO
		} else if strings.HasPrefix(mimeType, "image/") {
			itemType = ITEM_TYPE_IMAGE
		} else {
			itemType = ITEM_TYPE_OTHER
		}
	}
	return itemType
}
