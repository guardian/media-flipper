package initiator

import (
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/webapp/jobrunner"
	"net/http"
)

type InitiatorEndpoints struct {
	uploader UploadEndpointHandler
}

func NewInitiatorEndpoints(config *helpers.Config, redisClient *redis.Client, jobrunner *jobrunner.JobRunner) InitiatorEndpoints {
	return InitiatorEndpoints{
		uploader: UploadEndpointHandler{config: config, redisClient: redisClient, runner: jobrunner},
	}
}

func (e InitiatorEndpoints) WireUp(baseUrlPath string) {
	http.Handle(baseUrlPath+"/upload", e.uploader)
}
