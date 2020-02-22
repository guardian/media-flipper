package jobrunner

import (
	"errors"
	"github.com/davecgh/go-spew/spew"
	models2 "github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
)

func CreateTranscodeJob(jobDesc models2.JobStepTranscode, k8client *kubernetes.Clientset) error {
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform thumbnail with no media file")
		return errors.New("Can't perform thumbnail with no media file")
	}

	jsonTranscodeSettings, marshalErr := jobDesc.TranscodeSettings.InternalMarshalJSON()
	if marshalErr != nil {
		log.Printf("Could not convert settings into json: %s", marshalErr)
		log.Printf("Offending data was %s", spew.Sdump(jobDesc.TranscodeSettings))
		return marshalErr
	}
	vars := map[string]string{
		"WRAPPER_MODE":       "transcode",
		"JOB_CONTAINER_ID":   jobDesc.JobContainerId.String(),
		"JOB_STEP_ID":        jobDesc.JobStepId.String(),
		"FILE_NAME":          jobDesc.MediaFile,
		"TRANSCODE_SETTINGS": string(jsonTranscodeSettings),
		"MAX_RETRIES":        "10",
		"MEDIA_TYPE":         string(jobDesc.ItemType),
	}

	//jobName := fmt.Sprintf("mediaflipper-transcode-%s", path.Base(jobDesc.MediaFile))
	return CreateGenericJob(jobDesc.JobStepId, "flip-transc", vars, true, jobDesc.KubernetesTemplateFile, k8client)
}
