package jobrunner

import (
	"github.com/google/uuid"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"log"
)

func CreateGenericJob(jobStepID uuid.UUID, jobName string, envVars map[string]string, overwriteExistingVars bool, kubernetesTemplateFile string, k8client *kubernetes.Clientset) error {
	svcClient, svcClientErr := GetServiceClient(k8client)
	if svcClientErr != nil {
		log.Printf("ERROR: Could not get k8s service client: %s", svcClientErr)
		return svcClientErr
	}

	svcUrlPtr, svcUrlErr := FindServiceUrl(svcClient)
	if svcUrlErr != nil {
		log.Print("Could not determine return url from k8 service: ", svcUrlErr)
		return svcUrlErr
	} else {
		jobClient, cliErr := GetJobClient(k8client)
		if cliErr != nil {
			log.Printf("Could not create analysis job: %s", cliErr)
			return cliErr
		}

		envVars["WEBAPP_BASE"] = *svcUrlPtr
		return createGenericJobInternal(jobStepID, jobName, envVars, overwriteExistingVars, kubernetesTemplateFile, jobClient)
	}
}

func createGenericJobInternal(jobStepID uuid.UUID, jobName string, envVars map[string]string, overwriteExistingVars bool, kubernetesTemplateFile string, jobClient v1.JobInterface) error {
	jobPtr, loadErr := LoadFromTemplate(kubernetesTemplateFile)

	if loadErr != nil {
		log.Print("Could not load job template data: ", jobStepID)
		return loadErr
	}

	currentLabels := jobPtr.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}
	currentLabels["mediaflipper.jobStepId"] = jobStepID.String()
	jobPtr.SetLabels(currentLabels)

	vars := make([]v12.EnvVar, len(envVars))
	i := 0
	for k, v := range envVars {
		vars[i] = v12.EnvVar{Name: k, Value: v}
		i += 1
	}

	if !overwriteExistingVars {
		for _, v := range jobPtr.Spec.Template.Spec.Containers[0].Env {
			_, haveOverwrite := envVars[v.Name]
			if !haveOverwrite { //only re-add to the vars list if there is not one there already
				vars = append(vars, v)
			}
		}
	}

	jobPtr.Spec.Template.Spec.Containers[0].Env = vars

	jobPtr.ObjectMeta.Name = jobName

	_, err := jobClient.Create(jobPtr)
	if err != nil {
		log.Print("Can't create job: ", err)
		return err
	}

	return nil
}
