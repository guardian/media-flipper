package bulkprocessor

import (
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"net/http"
)

type ContentsHandler struct {
	redisClient *redis.Client
}

func (h ContentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	parsedUri, bulkListId, err := helpers.GetForId(r.RequestURI)
	if err != nil {
		log.Printf("Could not parse out url: %s", err)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not parse/extract url"}, w, 400)
		return
	}

	byStateRequest := parsedUri.Query().Get("state")
	byNameReqest := parsedUri.Query().Get("name")

	bulkList, bulkListErr := BulkListForId(*bulkListId, h.redisClient)
	if bulkListErr != nil {
		log.Printf("Could not retrieve bulk list from datastore: %s", bulkListErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not retrieve bulk list"}, w, 500)
		return
	}

	var recordsChan chan BulkItem
	var errChan chan error

	if byStateRequest == "" && byNameReqest == "" {
		//no names, retrieve everything
		recordsChan, errChan = bulkList.GetAllRecordsAsync(h.redisClient)
	} else if byStateRequest != "" && byNameReqest == "" {
		s := ItemStateFromString(byStateRequest)
		recordsChan, errChan = bulkList.FilterRecordsByStateAsync(s, h.redisClient)
	} else if byStateRequest == "" && byNameReqest != "" {
		recordsChan, errChan = bulkList.FilterRecordsByNameAsync(byNameReqest, h.redisClient)
	} else {
		s := ItemStateFromString(byStateRequest)
		recordsChan, errChan = bulkList.FilterRecordsByNameAndStateAsync(byNameReqest, s, h.redisClient)
	}

	//stream out the content as NDJSON.  We write the header here and then marshal and write each record as we receive it
	w.Header().Add("Content-Type", "application/x-ndjson")
	w.WriteHeader(200)

	func() {
		for {
			select {
			case record := <-recordsChan:
				if record == nil {
					return //we are done
				}
				content, marshalErr := json.Marshal(record)
				if marshalErr != nil {
					log.Printf("ERROR: Could not format data: %s. Offending data was %s", marshalErr, spew.Sdump(record))
				} else {
					_, writeErr := w.Write(content)
					if writeErr != nil {
						log.Printf("Could write outgoing data, continuing to next record: %s", writeErr)
					}
					w.Write([]byte("\n")) //newline delimited!!
				}
			case err := <-errChan:
				if err != nil {
					log.Printf("ERROR: content retrieval failed, returned content will be short: %s", err)
					return
				}
			}
		}
	}()
}
