package jobrunner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/guardian/mediaflipper/webapp/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

func CreateTranscodeJob(jobDesc models.JobStepTranscode, k8client *kubernetes.Clientset) error {
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform thumbnail with no media file")
		return errors.New("Can't perform thumbnail with no media file")
	}

	jsonTranscodeSettings, marshalErr := json.Marshal(jobDesc.TranscodeSettings)
	if marshalErr != nil {
		log.Printf("Could not convert settings into json: %s", marshalErr)
		log.Printf("Offending data was %s", spew.Sdump(jobDesc.TranscodeSettings))
	}
	vars := map[string]string{
		"WRAPPER_MODE":       "transcode",
		"JOB_CONTAINER_ID":   jobDesc.JobContainerId.String(),
		"JOB_STEP_ID":        jobDesc.JobStepId.String(),
		"FILE_NAME":          jobDesc.MediaFile,
		"TRANSCODE_SETTINGS": string(jsonTranscodeSettings),
		"MAX_RETRIES":        "10",
	}

	jobName := fmt.Sprintf("mediaflipper-transcode-%s", path.Base(jobDesc.MediaFile))
	return CreateGenericJob(jobDesc.JobStepId, jobName, vars, jobDesc.KubernetesTemplateFile, k8client)
}
