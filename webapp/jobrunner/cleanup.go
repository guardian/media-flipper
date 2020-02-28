package jobrunner

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
	"io"
	batchapi "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

/**
extract the logs for the pod identified by the `podInfo` parameter and return as a string.
this kinda assumes that the logs are not huge, i.e. not tens/hundreds of megs in size
*/
func extractLogs(podInfo *v1.Pod, podClient corev1.PodInterface) (string, error) {
	opts := v1.PodLogOptions{}
	req := podClient.GetLogs(podInfo.Name, &opts)
	podLogStream, streamErr := req.Stream()
	if streamErr != nil {
		log.Printf("ERROR extractLogs could not open log stream: %s", streamErr)
		return "", streamErr
	}
	defer podLogStream.Close()
	buf := new(bytes.Buffer)
	_, copyErr := io.Copy(buf, podLogStream)
	if copyErr != nil {
		log.Printf("ERROR extractLogs could not stream log content for %s: %s", podInfo.Name, copyErr)
	}
	return buf.String(), nil
}

/**
gets the logs for all pods associated with the given job id
concatenates them and returns the whole lot as a string, deleting the pods as it goes
this kinda assumes that the logs are not huge, i.e. not tens/hundreds of megs in size
*/
func getPodLogsAndRemove(jobID types.UID, podClient corev1.PodInterface) (string, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("controller-uid=%s", jobID),
	}
	podList, listErr := podClient.List(listOpts)
	if listErr != nil {
		log.Printf("ERROR getPodLogs could not list pods for job %s: %s", jobID, listErr)
		return "", listErr
	}

	var content string

	for _, podInfo := range podList.Items {
		logContent, getLogErr := extractLogs(&podInfo, podClient)
		if getLogErr != nil {
			log.Printf("ERROR getPodLogsAndRemove could not get pod logs: %s", getLogErr)
		} else {
			content += logContent
			//policy := metav1.DeletePropagationBackground

			//delOpts := metav1.DeleteOptions{
			//	TypeMeta:           metav1.TypeMeta{},
			//	PropagationPolicy:  &policy,
			//}
			//delErr := podClient.Delete(podInfo.Name, &delOpts)
			//if delErr != nil {
			//	log.Printf("ERROR getPodLogsAndRemove could not delete pod %s: %s", podInfo.Name, delErr)
			//}
		}
	}
	return content, nil
}

/**
extract the logs for the given job (provided by k8 job object) then delete both the job and the constituent pods
returns any error encountered or nil if the operation was successful
*/
func cleanUpK8Job(job *batchapi.Job, jobClient batchv1.JobInterface, podClient corev1.PodInterface, redisClient redis.Cmdable, outputKey string) error {
	logString, getLogErr := getPodLogsAndRemove(job.UID, podClient)
	if getLogErr != nil {
		log.Printf("ERROR cleanUpK8Job could not get logs: %s", getLogErr)
		return getLogErr
	}

	_, setErr := redisClient.Set(outputKey, logString, -1).Result()
	if setErr != nil {
		log.Printf("ERROR cleanUpK8Job could not store log content: %s", setErr)
		return setErr
	}

	policy := metav1.DeletePropagationBackground

	delOpts := metav1.DeleteOptions{
		TypeMeta:          metav1.TypeMeta{},
		PropagationPolicy: &policy,
	}

	deleteErr := jobClient.Delete(job.Name, &delOpts)
	if deleteErr != nil {
		log.Printf("ERROR cleanUpK8Job could not delete completed job: %s", deleteErr)
		return deleteErr
	}

	return nil
}

/**
try to find the k8 job with the given mediaflipper id
returns a pointer to the Job instance if one is found, nil if it is not found and an error if an error occurs
*/
func FindK8Job(mediaFlipperStepId uuid.UUID, jobclient batchv1.JobInterface) (*batchapi.Job, error) {
	p := 0
	continueToken := ""

	for {
		listOpts := metav1.ListOptions{
			LabelSelector: "mediaflipper.jobStepId",
			Continue:      continueToken,
		}

		result, err := jobclient.List(listOpts)
		if err != nil {
			log.Printf("ERROR FindK8Job could no list jobs: %s", err)
			return nil, err
		}

		idString := mediaFlipperStepId.String()
		for _, j := range result.Items {
			log.Printf("DEBUG FindK8Job checking %s with labels %s against %s", j.Name, j.Labels, idString)
			if j.Labels["mediaflipper.jobStepId"] == idString {
				log.Printf("DEBUG FindK8Job got job %s for mediaflipper %s", j.Name, idString)
				//make a copy seperate to the list and return a pointer to that, so entire list can be GC'd promptly
				rtn := j
				return &rtn, nil
			}
		}
		log.Printf("DEBUG FindK8Job no results found in page %d", p)
		p += 1
		if result.Continue == "" {
			log.Printf("DEBUG FindK8Job no more pages to check")
			return nil, nil
		}
		continueToken = result.Continue
	}
}

/**
master function. attempts Kubernetes cleanup of the given jobstep, by:
- finding the associated K8 job
- finding the pods associated with the given job
- extract the logs from the pods and delete them
- delete the K8 job
returns an error if any operations fail, or nil if successful.
*/
func CleanUpJobStep(step *models.JobStep, jobclient batchv1.JobInterface, podClient corev1.PodInterface, redisClient redis.Cmdable) error {
	if step == nil {
		return errors.New("passed jobstep was nil")
	}
	logsKey := fmt.Sprintf("mediaflipper:containerlog:%s", (*step).StepId().String())

	k8Job, jobErr := FindK8Job((*step).StepId(), jobclient)
	if jobErr != nil {
		log.Printf("ERROR CleanUpJobStep could not list K8 jobs: %s", jobErr)
		return jobErr
	}

	if k8Job == nil {
		log.Printf("ERROR CleanUpJobStep could not find a K8 job for job container id %s", (*step).ContainerId())
		return errors.New("could not find k8 job")
	}

	return cleanUpK8Job(k8Job, jobclient, podClient, redisClient, logsKey)
}
