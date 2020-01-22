package initiator

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type InitiatorEndpoints struct {
	uploader UploadEndpointHandler
}

func NewInitiatorEndpoints(config *helpers.Config, redisClient *redis.Client, k8client *kubernetes.Clientset) InitiatorEndpoints {
	return InitiatorEndpoints{
		uploader: UploadEndpointHandler{config: config, redisClient: redisClient, k8Client: k8client},
	}
}

func (e InitiatorEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/upload", e.uploader)
}
