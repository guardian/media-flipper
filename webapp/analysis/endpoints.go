package analysis

import (
	"github.com/go-redis/redis/v7"
	"net/http"
)

type AnalysisEndpoints struct {
	receiveData ReceiveData
	getData     GetData
}

func NewAnalysisEndpoints(client *redis.Client) AnalysisEndpoints {
	return AnalysisEndpoints{
		receiveData: ReceiveData{redisClient: client},
		getData:     GetData{redisClient: client},
	}
}

func (e AnalysisEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/result", e.receiveData)
	http.Handle(baseUrl+"/get", e.getData)
}
