package jobrunner

import (
	"errors"
	"fmt"
	"github.com/guardian/mediaflipper/webapp/models"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

func CreateThumbnailJob(jobDesc models.JobStepThumbnail, k8client *kubernetes.Clientset) error {
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform thumbnail with no media file")
		return errors.New("Can't perform thumbnail with no media file")
	}

	jobClient, cliErr := GetJobClient(k8client)
	if cliErr != nil {
		log.Printf("Could not create analysis job: %s", cliErr)
		return cliErr
	}

	jobPtr, loadErr := LoadFromTemplate(jobDesc.KubernetesTemplateFile)

	if loadErr != nil {
		log.Print("Could not load analysis job template data: ", loadErr)
		return loadErr
	}

	svcUrlPtr, svcUrlErr := FindServiceUrl(k8client)
	if svcUrlErr != nil {
		log.Print("Could not determine return url from k8 service: ", svcUrlErr)
	}

	currentLabels := jobPtr.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}
	currentLabels["mediaflipper.jobStepId"] = jobDesc.JobStepId.String()
	jobPtr.SetLabels(currentLabels)

	jobPtr.Spec.Template.Spec.Containers[0].Env = []v12.EnvVar{
		{Name: "WRAPPER_MODE", Value: "thumbnail"},
		{Name: "JOB_CONTAINER_ID", Value: jobDesc.JobContainerId.String()},
		{Name: "JOB_STEP_ID", Value: jobDesc.JobStepId.String()},
		{Name: "FILE_NAME", Value: jobDesc.MediaFile},
		{Name: "WEBAPP_BASE", Value: *svcUrlPtr},
		{Name: "MAX_RETRIES", Value: "10"},
	}
	jobPtr.ObjectMeta.Name = fmt.Sprintf("mediaflipper-thumbnail-%s", path.Base(jobDesc.MediaFile))

	if jobPtr.Spec.Template.Spec.RestartPolicy == "" {
		jobPtr.Spec.Template.Spec.RestartPolicy = "Never"
	}

	_, err := jobClient.Create(jobPtr)
	if err != nil {
		log.Print("Can't create analysis job: ", err)
		return err
	}

	return nil
}
