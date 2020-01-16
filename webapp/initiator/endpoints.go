package initiator

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/helpers"
	"net/http"
)

type InitiatorEndpoints struct {
	uploader UploadEndpointHandler
}

func NewInitiatorEndpoints(config *helpers.Config, redisClient *redis.Client) InitiatorEndpoints {
	return InitiatorEndpoints{
		uploader: UploadEndpointHandler{config: config, redisClient: redisClient},
	}
}

func (e InitiatorEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/upload", e.uploader)
}
