package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type SubmitHandler struct {
	redisClient        *redis.Client
	jobTemplateManager *models.JobTemplateManager
}

func (h SubmitHandler) newJobForItem(i BulkItem, templateId uuid.UUID) (*models.JobContainer, error) {
	job, err := h.jobTemplateManager.NewJobContainer(templateId)
	if err != nil {
		return nil, err
	}

	job.IncomingMediaFile = i.GetSourcePath()
	bulkItemId := i.GetId()
	job.AssociatedBulkItem = &bulkItemId
	return job, nil
}

func (h SubmitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "POST") {
		return
	}

}
