package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/bulk_models"
	"github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type BulkEndpoints struct {
	GetHandler            GetHandler
	UploadHandler         BulkListUploader
	ListHandler           ListHandler
	ContentsHandler       ContentsHandler
	UpdateHandler         UpdateHandler
	DeleteHandler         DeleteHandler
	RemoveDotFiles        RemoveDotFiles
	RemoveNonTranscodable RemoveNonTranscodableHandler
}

func NewBulkEndpoints(redisClient *redis.Client, templateManager *models.JobTemplateManager) BulkEndpoints {
	dao := bulk_models.BulkListDAOImpl{}

	return BulkEndpoints{
		GetHandler:            GetHandler{redisClient: redisClient},
		UploadHandler:         BulkListUploader{redisClient: redisClient},
		ListHandler:           ListHandler{redisClient: redisClient},
		ContentsHandler:       ContentsHandler{redisClient: redisClient},
		UpdateHandler:         UpdateHandler{redisClient: redisClient},
		DeleteHandler:         DeleteHandler{redisClient: redisClient},
		RemoveDotFiles:        RemoveDotFiles{redisClient: redisClient, dao: dao},
		RemoveNonTranscodable: RemoveNonTranscodableHandler{redisClient: redisClient, dao: dao},
	}
}

func (e BulkEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/get", e.GetHandler)
	http.Handle(baseUrl+"/upload", e.UploadHandler)
	http.Handle(baseUrl+"/list", e.ListHandler)
	http.Handle(baseUrl+"/content", e.ContentsHandler)
	http.Handle(baseUrl+"/update", e.UpdateHandler)
	http.Handle(baseUrl+"/delete", e.DeleteHandler)
	http.Handle(baseUrl+"/action/removeDotFiles", e.RemoveDotFiles)
	http.Handle(baseUrl+"/action/removeNonTranscodable", e.RemoveNonTranscodable)
}
