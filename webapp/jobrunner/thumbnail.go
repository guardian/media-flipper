package jobrunner

import (
	"errors"
	"fmt"
	"github.com/guardian/mediaflipper/webapp/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

func CreateThumbnailJob(jobDesc models.JobStepThumbnail, k8client *kubernetes.Clientset) error {
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform thumbnail with no media file")
		return errors.New("can't perform thumbnail with no media file")
	}

	var thumbFrameSeconds float64

	vars := map[string]string{
		"WRAPPER_MODE":     "thumbnail",
		"JOB_CONTAINER_ID": jobDesc.JobContainerId.String(),
		"JOB_STEP_ID":      jobDesc.JobStepId.String(),
		"FILE_NAME":        jobDesc.MediaFile,
		"THUMBNAIL_FRAME":  fmt.Sprintf("%f", thumbFrameSeconds),
		"MAX_RETRIES":      "10",
	}

	jobName := fmt.Sprintf("mediaflipper-thumbnail-%s", path.Base(jobDesc.MediaFile))
	return CreateGenericJob(jobDesc.JobStepId, jobName, vars, jobDesc.KubernetesTemplateFile, k8client)
}
