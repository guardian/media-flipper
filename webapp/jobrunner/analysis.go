package jobrunner

import (
	"errors"
	"github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
)

/**
create an analysis job based on the provided template
*/
func CreateAnalysisJob(jobDesc models.JobStepAnalysis, maybeOutPath string, k8client *kubernetes.Clientset) error {
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform analysis with no media file")
		return errors.New("Can't perform analysis with no media file")
	}

	vars := map[string]string{
		"WRAPPER_MODE":     "analyse",
		"JOB_CONTAINER_ID": jobDesc.JobContainerId.String(),
		"JOB_STEP_ID":      jobDesc.JobStepId.String(),
		"FILE_NAME":        jobDesc.MediaFile,
		"MAX_RETRIES":      "10",
		"MEDIA_TYPE":       string(jobDesc.ItemType),
		"OUTPUT_PATH":      maybeOutPath,
	}

	return CreateGenericJob(jobDesc.JobStepId, "flip-analysis", vars, true, jobDesc.KubernetesTemplateFile, k8client)
}
