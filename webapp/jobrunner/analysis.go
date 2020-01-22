package jobrunner

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/guardian/mediaflipper/webapp/models"
	"io/ioutil"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"path"
	"reflect"
)

/**
Loads up template data for an analysis job
*/
func LoadFromTemplate(fileName string) (*v1.Job, error) {
	bytes, readErr := ioutil.ReadFile(fileName)
	if readErr != nil {
		return nil, readErr
	}
	//THIS is the right way to read k8s manifests.... https://github.com/kubernetes/client-go/issues/193
	decode := scheme.Codecs.UniversalDeserializer()

	obj, groupVersionKind, err := decode.Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	log.Print("DEBUG: groupVersionKind is ", groupVersionKind)

	switch obj.(type) {
	case *v1.Job:
		return obj.(*v1.Job), nil
	default:
		log.Printf("Expected to get a job from template %s but got %s instead", fileName, reflect.TypeOf(obj).String())
		return nil, errors.New("Wrong manifest type")
	}
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
