package jobrunner

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/guardian/mediaflipper/webapp/models"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"path"
)

/**
Loads up template data for an analysis job
*/
func LoadFromTemplate(fileName string) (*v1.Job, error) {
	bytes, readErr := ioutil.ReadFile(fileName)
	if readErr != nil {
		return nil, readErr
	}

	var templateDesc v1.Job
	decodErr := yaml.Unmarshal(bytes, &templateDesc)
	if decodErr != nil {
		return nil, decodErr
	}
	return &templateDesc, nil
}

/**
create an analysis job based on the provided template
*/
func CreateAnalysisJob(jobDesc models.JobEntry, k8client *kubernetes.Clientset) error {
	log.Printf("In CreateAnalysisJob")
	if jobDesc.MediaFile == "" {
		log.Printf("Can't perform analysis with no media file")
		return errors.New("Can't perform analysis with no media file")
	}

	jobClient, cliErr := GetJobClient(k8client)
	if cliErr != nil {
		log.Printf("Could not create analysis job: %s", cliErr)
		return cliErr
	}

	jobPtr, loadErr := LoadFromTemplate("config/AnalysisJobTemplate.yaml")

	if loadErr != nil {
		log.Printf("Could not load analysis job template data: ", loadErr)
		return loadErr
	}

	spew.Dump(jobPtr.Spec.Template.Spec)
	spew.Dump(jobPtr.Spec.Template.Spec.RestartPolicy)

	svcUrlPtr, svcUrlErr := FindServiceUrl(k8client)
	if svcUrlErr != nil {
		log.Print("Could not determine return url from k8 service: ", svcUrlErr)
	}

	currentLabels := jobPtr.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}
	currentLabels["mediaflipper.jobId"] = jobDesc.JobId.String()
	jobPtr.SetLabels(currentLabels)

	jobPtr.Spec.Template.Spec.Containers[0].Env = []v12.EnvVar{
		{Name: "WRAPPER_MODE", Value: "analyse"},
		{Name: "JOB_ID", Value: jobDesc.JobId.String()},
		{Name: "FILE_NAME", Value: jobDesc.MediaFile},
		{Name: "WEBAPP_BASE", Value: *svcUrlPtr},
		{Name: "MAX_RETRIES", Value: "10"},
	}
	jobPtr.ObjectMeta.Name = fmt.Sprintf("mediaflipper-analysis-%s", path.Base(jobDesc.MediaFile))

	spew.Dump(jobPtr.Spec.Template.Spec)
	spew.Dump(jobPtr.Spec.Template.Spec.RestartPolicy)

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
