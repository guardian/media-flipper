package transcode

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"io/ioutil"
	"log"
	"net/http"
)

type ReceiveProgress struct {
	redisClient *redis.Client
}

func (h ReceiveProgress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

	rawContent, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		log.Print("ERROR: Could not read in content for progress update: ", readErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not read in content"}, w, 500)
		return
	}

	var progressUpdate models.TranscodeProgress
	marshalErr := json.Unmarshal(rawContent, &progressUpdate)
	if marshalErr != nil {
		log.Printf("ERROR: could not understand request body: %s. Offending data was %s.", marshalErr, string(rawContent))
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "could not understand request body"}, w, 400)
		return
	}

	emptyUuid := uuid.UUID{}
	if progressUpdate.JobContainerId == emptyUuid || progressUpdate.JobStepId == emptyUuid {
		log.Printf("job id parameters were empty, can't store")
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", "job id parameters were empty, can't store"}, w, 400)
		return
	}

	dataKey := fmt.Sprintf("mediaflipper:jobprogress:%s", progressUpdate.JobStepId.String())
	h.redisClient.ZAdd(dataKey, &redis.Z{
		Score:  float64(progressUpdate.Timestamp),
		Member: string(rawContent), //no point re-marshalling the object as we know it's good if we got here
	})
	helpers.WriteJsonContent(helpers.GenericErrorResponse{"ok", "data stored"}, w, 201)
}
