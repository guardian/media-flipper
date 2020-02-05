package transcode

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type TranscodeEndpoints struct {
	receiveData     ReceiveData
	receiveProgress ReceiveProgress
	getProgress     GetProgress
}

func NewTranscodeEndpoints(redisClient *redis.Client) TranscodeEndpoints {
	return TranscodeEndpoints{
		receiveData:     ReceiveData{redisClient: redisClient},
		receiveProgress: ReceiveProgress{redisClient: redisClient},
		getProgress:     GetProgress{redisClient: redisClient},
	}
}

func (t TranscodeEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/result", t.receiveData)
	http.Handle(baseUrl+"/newprogress", t.receiveProgress)
	http.Handle(baseUrl+"/progress", t.getProgress)
}
