package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
	"strings"
	"time"
	//"golang.org/x/text/encoding/charmap"
)

type BulkListUploader struct {
	redisClient *redis.Client
}

func (h BulkListUploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	uid, _ := uuid.NewRandom()
	newBulk := BulkListImpl{
		BulkListId:   uid,
		CreationTime: time.Now(),
	}

	//_ := charmap.ISO8859_1.NewDecoder()
	rawLinesChan, rawLinesErrChan := AsyncNewlineReader(r.Body, nil, 10)

	completedChan := make(chan error)

	go asyncInputProcessor(&newBulk, completedChan, rawLinesChan, rawLinesErrChan, h.redisClient)

	processingErrored := <-completedChan
	storeErr := newBulk.Store(h.redisClient)
	if storeErr != nil {
		log.Print("Could not write bulk data: ", storeErr)
		if processingErrored == nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not write new batch"}, w, 500)
			return
		}
	}

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
testing, frontend has been reporting duped uuids; is this in the retrieval or setting? this will tell us.
*/
func guaranteedGoodItem(linePtr *string, bulkList BulkList, redisClient redis.Cmdable) (BulkItem, error) {
	newItem := NewBulkItem(*linePtr, -1)
	alreadyExists, existErr := bulkList.ExistsInIndex(newItem.GetId(), redisClient)
	if existErr != nil {
		log.Printf("ERROR: unable to check index!")
		return nil, existErr
	}
	if alreadyExists {
		return guaranteedGoodItem(linePtr, bulkList, redisClient)
	} else {
		return newItem, nil
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
			//log.Printf("Got %s", spew.Sdump(linePtr))
			if linePtr == nil {
				completedChan <- nil
				return
			} else {
				trimmedFilename := strings.TrimSpace(*linePtr)
				if len(trimmedFilename) > 1 && !strings.HasPrefix(trimmedFilename, "#") {
					newItem, checkErr := guaranteedGoodItem(linePtr, bulkList, redisClient)
					if checkErr != nil {
						completedChan <- checkErr
						return
					}
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
}
