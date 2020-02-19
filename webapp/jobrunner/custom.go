package jobrunner

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

func CreateCustomJob(jobDesc models.JobStepCustom, container *models.JobContainer, k8client *kubernetes.Clientset, redisClient redis.Cmdable) error {
	var transcodedMediaPath string
	if container.TranscodedMediaId != nil {
		fileEntry, getErr := models.FileEntryForId(*container.TranscodedMediaId, redisClient)
		if getErr != nil {
			log.Printf("ERROR: Could not get a file entry for id %s", *container.TranscodedMediaId)
		} else {
			transcodedMediaPath = fileEntry.ServerPath
		}
	}

	var thumbnailImagePath string
	if container.ThumbnailId != nil {
		fileEntry, getErr := models.FileEntryForId(*container.ThumbnailId, redisClient)
		if getErr != nil {
			log.Printf("ERROR: Could not get a file entry for id %s", *container.ThumbnailId)
		} else {
			thumbnailImagePath = fileEntry.ServerPath
		}
	}

	vars := map[string]string{
		"WRAPPER_MODE":     "custom",
		"JOB_CONTAINER_ID": jobDesc.JobContainerId.String(),
		"JOB_STEP_ID":      jobDesc.JobStepId.String(),
		"FILE_NAME":        jobDesc.MediaFile,
		"MAX_RETRIES":      "10",
		"MEDIA_TYPE":       string(jobDesc.ItemType),
		"TRANSCODED_MEDIA": transcodedMediaPath,
		"THUMBNAIL_IMAGE":  thumbnailImagePath,
	}

	for k, v := range jobDesc.CustomArguments { //add in custom arguments
		vars[k] = v
	}

	jobName := fmt.Sprintf("mediaflipper-custom-%s", path.Base(jobDesc.MediaFile))

	return CreateGenericJob(jobDesc.JobStepId, jobName, vars, false, jobDesc.KubernetesTemplateFile, k8client)
}
