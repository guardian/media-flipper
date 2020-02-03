package transcode

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type TranscodeEndpoints struct {
	receiveData ReceiveData
}

func NewTranscodeEndpoints(redisClient *redis.Client) TranscodeEndpoints {
	return TranscodeEndpoints{
		receiveData: ReceiveData{redisClient: redisClient},
	}
}

func (t TranscodeEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/result", t.receiveData)
}
