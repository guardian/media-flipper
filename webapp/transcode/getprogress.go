package transcode

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
	"strconv"
)

type GetProgress struct {
	redisClient *redis.Client
}

func (h GetProgress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUri, jobStepId, paramsErr := helpers.GetForId(r.RequestURI)
	if paramsErr != nil {
		helpers.WriteJsonContent(paramsErr, w, 400)
	}

	var countSteps int64
	if countStepsString := requestUri.Query().Get("count"); countStepsString != "" {
		var parseErr error
		countSteps, parseErr = strconv.ParseInt(countStepsString, 10, 64)
		if parseErr != nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "count parameter must be an integer number"}, w, 400)
			return
		}
	} else {
		countSteps = 1
	}

	dataKey := fmt.Sprintf("mediaflipper:jobprogress:%s", jobStepId.String())

	response, getErr := h.redisClient.ZRevRange(dataKey, 0, countSteps).Result()
	if getErr != nil {
		log.Printf("ERROR: Could not scan progress data for %s: %s", dataKey, getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"db_error", "could not scan progress data for this job"}, w, 500)
		return
	}

	count, _ := h.redisClient.ZCard(dataKey).Result()

	steps := make([]models.TranscodeProgress, len(response))
	for i, jsonBlob := range response {
		marshalErr := json.Unmarshal([]byte(jsonBlob), &steps[i])
		if marshalErr != nil {
			log.Printf("WARNING: invalid data in datastore! %s. Offending data was %s", marshalErr, jsonBlob)
		}
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"count":   count,
		"entries": steps,
	}, w, 200)
}
