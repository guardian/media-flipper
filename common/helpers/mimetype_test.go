package helpers

import "testing"

func TestItemTypeForFilepath(t *testing.T) {
	//ItemTypeForFilepath should return ITEM_TYPE_VIDEO for a known video extension
	result := ItemTypeForFilepath("path/to/some/video.mp4")
	if result != ITEM_TYPE_VIDEO {
		t.Errorf("ItemTypeForFilepath returned incorrect type %s for mp4", result)
	}

	//ItemTypeForFilepath should return ITEM_TYPE_AUDIO for a known audio extension
	aResult := ItemTypeForFilepath("path/to/some/audio.mp3")
	if aResult != ITEM_TYPE_AUDIO {
		t.Errorf("ItemTypeForFilepath returned incorrect type %s for mp3", result)
	}

	iResult := ItemTypeForFilepath("path/to/some/image.jpg")
	if iResult != ITEM_TYPE_IMAGE {
		t.Errorf("ItemTypeForFilepath returned incorrect type %s for jpg", result)
	}

	oResult := ItemTypeForFilepath("path/to/some/meta.xml")
	if oResult != ITEM_TYPE_OTHER {
		t.Errorf("ItemTypeForFilepath returned incorrect type %s for xml", result)
	}
}
