package jobrunner

// see https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go

import (
	"errors"
	"fmt"
	_ "fmt"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"time"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func InClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Could not establish cluster connection: ", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Could not establish cluster connection: ", err)
		return nil, err
	}

	return clientset, nil
}

func GetMyNamespace(client *kubernetes.Clientset) (string, error) {
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

func safeStartTimeString(timeval *metav1.Time) string {
	if timeval == nil {
		return ""
	} else {
		return timeval.Format(time.RFC3339)
	}
}

func FindRunnerFor(jobId uuid.UUID, client *kubernetes.Clientset) (*[]models.JobRunnerDesc, error) {
	ns, nsErr := GetMyNamespace(client)
	if nsErr != nil {
		return nil, nsErr
	}

	listOpts := metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("mediaflipper.jobId=%s", jobId),
		Watch:          false,
		TimeoutSeconds: nil,
		Limit:          0,
	}

	response, err := client.BatchV1().Jobs(ns).List(listOpts)
	if err != nil {
		log.Print("ERROR: Could not list k8s job containers: ", err)
		return nil, err
	}

	rtn := make([]models.JobRunnerDesc, len(response.Items))
	for i, jobDesc := range response.Items {
		log.Printf("Got job name %s in status %s with labels %s", jobDesc.Name, jobDesc.Status, jobDesc.Labels)
		var statusString string
		if jobDesc.Status.Failed == 0 && jobDesc.Status.Succeeded == 0 {
			statusString = "InProgress"
		} else if jobDesc.Status.Failed > 0 && jobDesc.Status.Succeeded == 0 {
			statusString = "Failed"
		} else if jobDesc.Status.Failed == 0 && jobDesc.Status.Succeeded > 0 {
			statusString = "Success"
		} else {
			statusString = "Unknown"
		}

		rtn[i] = models.JobRunnerDesc{
			JobUID:         string(jobDesc.UID),
			Status:         statusString,
			StartTime:      safeStartTimeString(jobDesc.Status.StartTime),
			CompletionTime: safeStartTimeString(jobDesc.Status.CompletionTime),
			Name:           jobDesc.Name,
		}
	}
	return &rtn, nil
}
