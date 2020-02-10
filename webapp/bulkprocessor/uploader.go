package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
	"time"
)

type BulkListUploader struct {
	redisClient *redis.Client
}

func (h BulkListUploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	newBulk := BulkListImpl{
		BulkListId:   uuid.New(),
		CreationTime: time.Now(),
	}
	rawLinesChan, rawLinesErrChan := AsyncNewlineReader(r.Body, 10)

	completedChan := make(chan error)

	go asyncInputProcessor(&newBulk, completedChan, rawLinesChan, rawLinesErrChan, h.redisClient)

	processingErrored := <-completedChan
	if processingErrored != nil {
		log.Printf("could not process incoming data: %s", processingErrored)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: processingErrored.Error(),
		}, w, 500)
	} else {
		helpers.WriteJsonContent(map[string]string{"status": "ok", "detail": "created bulk", "bulkid": newBulk.BulkListId.String()}, w, 200)
	}
}

/**
async goroutine that receives data from either a stream of file-lines or its corresponding error channel and adds content
to the given BulkList
*/
func asyncInputProcessor(bulkList BulkList, completedChan chan error, rawLinesChan chan *string, rawLinesErrChan chan error, redisClient *redis.Client) {
	for {
		select {
		case readErr := <-rawLinesErrChan:
			log.Printf("ERROR reading from stream: %s", readErr)
			completedChan <- readErr
			return
		case linePtr := <-rawLinesChan:
			if linePtr == nil {
				completedChan <- nil
				return
			} else {
				newItem := NewBulkItem(*linePtr, -1)
				addErr := bulkList.AddRecord(newItem, redisClient)
				if addErr != nil {
					log.Printf("Could not add new item to bulk: %s", addErr)
					completedChan <- addErr
					return
				}
			}
		}
	}
}
