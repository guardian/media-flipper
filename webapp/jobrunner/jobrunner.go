package jobrunner

import (
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"time"
)

type JobRunner struct {
	redisClient     *redis.Client
	k8client        *kubernetes.Clientset
	shutdownChan    chan bool
	queuePollTicker *time.Ticker
	maxJobs         int32
}

/**
create a new JobRunner object
*/
func NewJobRunner(redisClient *redis.Client, k8client *kubernetes.Clientset, channelBuffer int, maxJobs int32) JobRunner {
	shutdownChan := make(chan bool)
	queuePollTicker := time.NewTicker(1 * time.Second)

	runner := JobRunner{
		redisClient:     redisClient,
		k8client:        k8client,
		shutdownChan:    shutdownChan,
		queuePollTicker: queuePollTicker,
		maxJobs:         maxJobs,
	}
	go runner.requestProcessor()
	return runner
}

/**
add the provided JobEntry to the queue for processing
*/
func (j *JobRunner) AddJob(job *models.JobEntry, predefinedType string) error {
	newRecord := JobRunnerRequest{
		requestId:      uuid.New(),
		predefinedType: predefinedType,
		forJob:         *job,
	}

	return pushToRequestQueue(j.redisClient, &newRecord)
}

/**
goroutine to process incoming requests
*/
func (j *JobRunner) requestProcessor() {
	log.Print("Started requestProcessor routine")
	for {
		select {
		case <-j.queuePollTicker.C:
			log.Printf("DEBUG: JobRunner queue tick")
			j.clearCompletedTick()
			j.waitingQueueTick()
		}
	}
}

/**
trigger the action for a given item and put it onto the running queue if successful
*/
func (j *JobRunner) actionRequest(rq *JobRunnerRequest) error {
	if rq.predefinedType == "analysis" {
		err := CreateAnalysisJob(rq.forJob, j.k8client)
		if err != nil {
			log.Print("Could not create analysis job! ", err)
			return err
		}
		pushErr := pushToRunningQueue(j.redisClient, rq)
		if pushErr != nil {
			log.Printf("Could not update running queue! ", pushErr)
			return pushErr
		}
		return nil
	} else {
		log.Print("Other job types not yet implemented!")
		return errors.New("other job types not yet implemented")
	}
}

func (j *JobRunner) clearCompletedTick() {
	set, checkErr := checkQueueLock(j.redisClient, RUNNING_QUEUE)
	if checkErr != nil {
		log.Printf("Could not check running queue lock: %s", checkErr)
		return
	}

	if set {
		log.Printf("Running queue is locked, not performing clear completed")
		return
	}

	setQueueLock(j.redisClient, RUNNING_QUEUE)

	queueSnapshot, snapErr := copyRunningQueueContent(j.redisClient)
	if snapErr != nil {
		log.Printf("ERROR: Could not clear completed jobs, queue snapshot gave an error")
		return
	}

	for i, runningJob := range *queueSnapshot {
		var jobId uuid.UUID
		if runningJob.forJob.JobId.ID() != 0 {
			jobId = runningJob.forJob.JobId
		} else {
			jobId = runningJob.requestId
		}

		runners, runErr := FindRunnerFor(jobId, j.k8client)
		if runErr != nil {
			log.Print("Could not get runner for ", jobId, ": ", runErr)
			continue //proceed to next one, don't abort
		}

		for _, runner := range *runners {
			switch runner.Status {
			case models.CONTAINER_COMPLETED:
				runningJob.forJob.Status = models.JOB_COMPLETED
				putErr := models.PutJob(&runningJob.forJob, j.redisClient)
				if putErr != nil {
					log.Printf("Could not update job %s: %s", runningJob.forJob, putErr)
				} else {
					removeFromQueue(j.redisClient, RUNNING_QUEUE, int64(i))
				}
			case models.CONTAINER_FAILED:
				runningJob.forJob.Status = models.JOB_FAILED
				putErr := models.PutJob(&runningJob.forJob, j.redisClient)
				if putErr != nil {
					log.Printf("Could not update job %s: %s", runningJob.forJob, putErr)
				} else {
					removeFromQueue(j.redisClient, RUNNING_QUEUE, int64(i))
				}
			case models.CONTAINER_ACTIVE:
				runningJob.forJob.Status = models.JOB_STARTED
				putErr := models.PutJob(&runningJob.forJob, j.redisClient)
				if putErr != nil {
					log.Printf("Could not update job %s: %s", runningJob.forJob, putErr)
				} else {
					removeFromQueue(j.redisClient, RUNNING_QUEUE, int64(i))
				}
			}
		}
	}

	releaseQueueLock(j.redisClient, RUNNING_QUEUE)
}

/**
internal function to process items on the waiting queue, up until we either run out of items on the queue or have the
max running jobs
*/
func (j *JobRunner) waitingQueueTick() {
	for {
		queuelen, getErr := getRunningQueueLength(j.redisClient)
		if getErr != nil {
			log.Printf("ERROR: Could not get queue length: %s", getErr)
			continue
		}
		if queuelen >= int64(j.maxJobs) {
			log.Printf("Max running jobs reached")
			break
		}
		if queuelen == 0 {
			log.Print("End of queue reached")
			break
		}
		newJob, getErr := getNextRequestQueueEntry(j.redisClient)
		if getErr == nil {
			if newJob == nil {
				log.Printf("No more jobs to get")
				break
			} else {
				j.actionRequest(newJob)
			}
		} else {
			log.Printf("Could not get next job to process!")
			break
		}
	}
}
