package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	models2 "github.com/guardian/mediaflipper/common/models"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type ListJobHandler struct {
	RedisClient *redis.Client
}

type ListJobResponse struct {
	Status     string                  `json:"status"`
	NextCursor uint64                  `json:"nextCursor"`
	Entries    *[]models2.JobContainer `json:"entries"`
}

/**
list out job items in order of creation time

query parameters:
- startindex - the first item to get. Defaults to 0, i.e. the latet
- limit - the maximum number of items to get. Defaults to 100
- jobId - limit results to this specific job ID. Used when linking from batch view -> job view
*/
func (h ListJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)

	var windowStart int64
	windowStartString := requestUrl.Query().Get("startindex")
	if windowStartString == "" {
		windowStart = 0
	} else {
		var parseErr error
		windowStart, parseErr = strconv.ParseInt(windowStartString, 10, 64)
		if parseErr != nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "latest parameter must be a unix timestamp"}, w, 400)
			return
		}
	}

	var windowEnd int64
	windowEndString := requestUrl.Query().Get("limit")
	if windowEndString == "" {
		windowEnd = 100
	} else {
		var parseErr error
		windowEnd, parseErr = strconv.ParseInt(windowEndString, 10, 64)
		if parseErr != nil {
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "earliest parameter must be a unix timestamp"}, w, 400)
			return
		}
	}

	var jobs *[]models2.JobContainer
	var nextCursor uint64
	var getErr error
	specificIdString := requestUrl.Query().Get("jobId")
	bulkIdString := requestUrl.Query().Get("bulkItem")
	stateString := requestUrl.Query().Get("state")

	var statusArg *models2.JobStatus
	if stateString == "" {
		statusArg = nil
	} else {
		statusArgContent := models2.JobStatusFromString(stateString)
		statusArg = &statusArgContent
	}

	if specificIdString != "" {
		specificId, idParseErr := uuid.Parse(specificIdString)
		if idParseErr != nil {
			log.Printf("ERROR ListJobHandler was passed invalid jobId '%s': %s", specificIdString, idParseErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "jobId parameter not valid"}, w, 400)
			return
		}
		job, singleGetErr := models2.JobContainerForId(specificId, h.RedisClient)
		getErr = singleGetErr
		if job != nil {
			jobs = &[]models2.JobContainer{*job}
		} else {
			jobs = nil
		}
	} else if bulkIdString != "" {
		bulkId, idParseErr := uuid.Parse(bulkIdString)
		if idParseErr != nil {
			log.Printf("ERROR ListJobHandler was passed invalid bulkItem id '%s': %s", specificIdString, idParseErr)
			helpers.WriteJsonContent(helpers.GenericErrorResponse{"bad_data", "jobId parameter not valid"}, w, 400)
			return
		}
		jobsList, singleGetErr := models2.JobContainerForBulkItem(bulkId, h.RedisClient)
		getErr = singleGetErr
		jobs = &jobsList
	} else {
		jobs, nextCursor, getErr = models2.ListJobContainers(uint64(windowStart), windowEnd, h.RedisClient, models2.SORT_CTIME, statusArg)
	}

	if getErr != nil {
		log.Printf("ERROR ListJobHandler could not look up data: %s", getErr)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "could not get data, see logs for details",
		}, w, 500)
		return
	}
	response := ListJobResponse{
		Status:     "ok",
		NextCursor: nextCursor,
		Entries:    jobs,
	}

	helpers.WriteJsonContent(response, w, 200)
}
