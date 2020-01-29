package jobs

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type ListJobHandler struct {
	RedisClient *redis.Client
}

type ListJobResponse struct {
	Status     string                 `json:"status"`
	NextCursor uint64                 `json:"nextCursor"`
	Entries    *[]models.JobContainer `json:"entries"`
}

func (h ListJobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	jobs, nextCursor, getErr := models.ListJobContainers(0, 50, h.RedisClient)
	if getErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "db_error",
			Detail: "could not get data, see logs for details",
		}, w, 500)
		return
	}

	//finalJsonString := "[" + strings.Join(*jsonBlobs, ",") + "]"
	//helpers.WriteJsonContent(ListJobResponse{
	//	Status:     "ok",
	//	NextCursor: nextCursor,
	//	Entries:    finalJsonString,
	//}, w, 200)

	response := ListJobResponse{
		Status:     "ok",
		NextCursor: nextCursor,
		Entries:    jobs,
	}

	helpers.WriteJsonContent(response, w, 200)
}
