package jobrunner

// see https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	models2 "github.com/guardian/mediaflipper/common/models"
	"io/ioutil"
	v1batch "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	v13 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"reflect"
	"time"
	//
	// Uncomment to load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

type k8ClientWrapper interface {
	GetJobClient()
}

/**
initialise connection to Kubernetes from a pod within the cluster
*/
func InClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Print("Could not establish cluster connection: ", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Print("Could not establish cluster connection: ", err)
		return nil, err
	}

	return clientset, nil
}

/**
initialise a connection to Kubernetes from outside the cluster. This requires a kubeconfig file (e.g. for kubectl)
to describe how to connect and authorise to the cluster
*/
func OutOfClusterClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Print("Could not build out-of-cluster config: ", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Print("Could not establish cluster connection: ", err)
		return nil, err
	}

	return clientset, nil
}

/**
determine the namespace that we are running in.  This (obviously) assumes that it is running inside a cluster
*/
func GetMyNamespace() (string, error) {
	//if we are in the cluster then this file should be present
	_, statErr := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if statErr != nil {
		if os.IsNotExist(statErr) {
			log.Printf("Out of cluster configuration not implemented yet")
			return "", errors.New("Not implemented")
		}
		log.Print("ERROR asserting kubernetes namespace: ", statErr)
		return "", statErr
	}

	content, readErr := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if readErr != nil {
		log.Print("Could not read in k8s namespace: ", readErr)
		return "", readErr
	}
	return string(content), nil
}

/**
helper function to get a "Jobs" client from the clientset
*/
func GetJobClient(k8client *kubernetes.Clientset) (v1.JobInterface, error) {
	ns, nsErr := GetMyNamespace()
	if nsErr != nil {
		return nil, nsErr
	}

	jobClient := k8client.BatchV1().Jobs(ns)
	return jobClient, nil
}

func GetServiceClient(k8client *kubernetes.Clientset) (v13.ServiceInterface, error) {
	ns, nsErr := GetMyNamespace()
	if nsErr != nil {
		return nil, nsErr
	}

	return k8client.CoreV1().Services(ns), nil
}

/**
goes through a list of ServicePort descriptions to find the one matching the given name
returns a pointer to the port number if successful or nil if not.
*/
func findPortInList(portName string, in *[]v12.ServicePort) *int32 {
	for _, portDesc := range *in {
		if portDesc.Name == portName {
			return &portDesc.Port
		}
	}
	return nil
}

/**
Assuming the server is running inside the cluster, determine the URL to access it from by locating the
Service description
*/
func FindServiceUrl(serviceClient v13.ServiceInterface) (*string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: "app=webapp,stack=MediaFlipper",
	}

	response, listErr := serviceClient.List(listOpts)
	if listErr != nil {
		return nil, listErr
	}

	for _, serviceDesc := range response.Items {
		hostPart := serviceDesc.Name
		portNumPtr := findPortInList("webapp", &serviceDesc.Spec.Ports)
		if portNumPtr != nil {
			urlString := fmt.Sprintf("http://%s:%d", hostPart, *portNumPtr)
			return &urlString, nil
		}
	}
	return nil, errors.New("Could not determine url, either no services match expected labels or no `webapp` port defined")
}

/**
helper function to string-format a time that may be nil
returns the formatted time string if the timeval is valid or an empty string otherwise
*/
func safeStartTimeString(timeval *metav1.Time) string {
	if timeval == nil {
		return ""
	} else {
		return timeval.Format(time.RFC3339)
	}
}

/**
look up the Kubernetes Job associated with the given mediaflipper job ID
*/
func FindRunnerFor(jobId uuid.UUID, client v1.JobInterface) (*[]models2.JobRunnerDesc, error) {
	listOpts := metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("mediaflipper.jobStepId=%s", jobId),
		Watch:          false,
		TimeoutSeconds: nil,
		Limit:          0,
	}

	response, err := client.List(listOpts)
	if err != nil {
		log.Print("ERROR: Could not list k8s job containers: ", err)
		return nil, err
	}

	rtn := make([]models2.JobRunnerDesc, len(response.Items))
	for i, jobDesc := range response.Items {
		//log.Printf("Got job name %s in status %s with labels %s", jobDesc.Name, jobDesc.Status.String(), jobDesc.Labels)
		var statusVal models2.ContainerStatus
		cond := jobDesc.Status.Conditions
		if len(cond) > 0 && cond[0].Type == v1batch.JobFailed {
			statusVal = models2.CONTAINER_FAILED
		} else if jobDesc.Status.Failed == 0 && jobDesc.Status.Succeeded == 0 {
			statusVal = models2.CONTAINER_ACTIVE
		} else if jobDesc.Status.Failed > 0 && jobDesc.Status.Succeeded == 0 {
			statusVal = models2.CONTAINER_FAILED
		} else if jobDesc.Status.Succeeded > 0 {
			statusVal = models2.CONTAINER_COMPLETED
		} else if jobDesc.Status.Failed == 0 && jobDesc.Status.Succeeded == 0 && jobDesc.Status.Active == 0 { //no pods left!
			statusVal = models2.CONTAINER_FAILED
		} else {
			statusVal = models2.CONTAINER_UNKNOWN_STATE
		}

		rtn[i] = models2.JobRunnerDesc{
			JobUID:         string(jobDesc.UID),
			Status:         statusVal,
			StartTime:      safeStartTimeString(jobDesc.Status.StartTime),
			CompletionTime: safeStartTimeString(jobDesc.Status.CompletionTime),
			Name:           jobDesc.Name,
		}
	}
	return &rtn, nil
}

/**
Loads up template data for an analysis job
*/
func LoadFromTemplate(fileName string) (*v1batch.Job, error) {
	bytes, readErr := ioutil.ReadFile(fileName)
	if readErr != nil {
		return nil, readErr
	}
	//THIS is the right way to read k8s manifests.... https://github.com/kubernetes/client-go/issues/193
	decode := scheme.Codecs.UniversalDeserializer()

	obj, _, err := decode.Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	//log.Print("DEBUG: groupVersionKind is ", groupVersionKind)

	switch obj.(type) {
	case *v1batch.Job:
		return obj.(*v1batch.Job), nil
	default:
		log.Printf("Expected to get a job from template %s but got %s instead", fileName, reflect.TypeOf(obj).String())
		return nil, errors.New("Wrong manifest type")
	}
}
