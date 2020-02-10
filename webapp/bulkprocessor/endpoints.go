package bulkprocessor

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type BulkEndpoints struct {
	GetHandler    GetHandler
	UploadHandler BulkListUploader
}

func NewBulkEndpoints(redisClient *redis.Client) BulkEndpoints {
	return BulkEndpoints{
		GetHandler:    GetHandler{redisClient: redisClient},
		UploadHandler: BulkListUploader{redisClient: redisClient},
	}
}

func (e BulkEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/get", e.GetHandler)
	http.Handle(baseUrl+"/upload", e.UploadHandler)
}
