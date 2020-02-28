package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"io"
	"log"
	"net/http"
)

type GetLogsHandler struct {
	redisClient *redis.Client
}

func (h GetLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	qps, qpErr := helpers.GetQueryParams(r.RequestURI)
	if qpErr != nil {
		log.Printf("ERROR GetLogsHandler could not understand the passed url: %s", qpErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "could not understand the passed url"}, w, 400)
		return
	}

	stepId, sParseErr := uuid.Parse(qps.Get("stepId"))
	if sParseErr != nil {
		log.Printf("ERROR GetLogsHandler could not understand the jobId '%s': %s", qps.Get("stepId"), sParseErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "stepId not valid or missing"}, w, 400)
		return
	}

	stream, err := models.GetContainerLogContentStream(stepId, h.redisClient)
	if err != nil {
		log.Printf("ERROR GetLogsHandler could not retrieve logs: %s", err)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"error", err.Error()}, w, 500)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	_, streamErr := io.Copy(w, stream)
	if streamErr != nil {
		log.Printf("ERROR GetLogsHandler could not stream all content to client: %s", streamErr)
	}
}
