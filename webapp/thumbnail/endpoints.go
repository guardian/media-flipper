package thumbnail

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type ThumbnailEndpoints struct {
	receiveData ReceiveData
}

func NewThumbnailEndpoints(redisClient *redis.Client) ThumbnailEndpoints {
	return ThumbnailEndpoints{
		ReceiveData{redisClient: redisClient},
	}
}

func (e *ThumbnailEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/result", e.receiveData)
}
