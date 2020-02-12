package bulkprocessor

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type UpdateHandler struct {
	redisClient *redis.Client
}

func (h UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	_, bulkListId, reqErr := helpers.GetForId(r.RequestURI)
	if reqErr != nil {
		helpers.WriteJsonContent(reqErr, w, 400)
		return
	}

	bodyContent, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("BatchUpdateHandler could not read body content: %s", readErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not read content body"}, w, 500)
		return
	}

	var rq BulkListUpdate
	marshalErr := json.Unmarshal(bodyContent, &rq)
	if marshalErr != nil {
		log.Printf("Could not unmarshal content body: %s. Offending data was %s.", marshalErr, string(bodyContent))
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not understand content"}, w, 400)
		return
	}

	bulkList, getErr := BulkListForId(*bulkListId, h.redisClient)
	if getErr != nil {
		log.Printf("could not retrieve bulk list for id %s: %s", bulkListId, getErr)
		if strings.Contains(getErr.Error(), "redis: nil") {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no batch list with that id"}, w, 404)
		} else {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not get batch list"}, w, 500)
		}
		return
	}

	bulkList.SetNickName(rq.NickName)
	bulkList.SetTemplateId(rq.TemplateId)
	storErr := bulkList.Store(h.redisClient)
	if storErr != nil {
		log.Printf("could not store updated batch list %s: %s", bulkListId, storErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not store updated batch list"}, w, 500)
		return
	}
	helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "updated"}, w, 200)
}
