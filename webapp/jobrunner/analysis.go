package jobrunner

import (
	"errors"
	"fmt"
	models2 "github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

/**
create an analysis job based on the provided template
*/
func CreateAnalysisJob(jobDesc models2.JobStepAnalysis, k8client *kubernetes.Clientset) error {
	log.Printf("In CreateAnalysisJob")
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
	}

	jobName := fmt.Sprintf("mediaflipper-analysis-%s", path.Base(jobDesc.MediaFile))

	return CreateGenericJob(jobDesc.JobStepId, jobName, vars, jobDesc.KubernetesTemplateFile, k8client)
}
