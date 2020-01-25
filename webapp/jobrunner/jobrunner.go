package jobrunner

import (
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/models"
	"github.com/jinzhu/copier"
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
func (j *JobRunner) AddJob(container *models.JobContainer) error {
	result := pushToRequestQueue(j.redisClient, container)
	log.Printf("Enqueued job for processing: %s", container.Id)
	return result
}

/**
goroutine to process incoming requests
*/
func (j *JobRunner) requestProcessor() {
	log.Print("Started requestProcessor routine")
	for {
		select {
		case <-j.queuePollTicker.C:
			//log.Printf("DEBUG: JobRunner queue tick")
			j.clearCompletedTick()
			j.waitingQueueTick()
		}
	}
}

///**
//trigger the action for a given item and put it onto the running queue if successful
//*/
//func (j *JobRunner) actionRequest(rq *JobRunnerRequest) error {
//	if rq.PredefinedType == "analysis" {
//		err := CreateAnalysisJob(rq.ForJob, j.k8client)
//		if err != nil {
//			log.Print("Could not create analysis job! ", err)
//			return err
//		}
//		log.Printf("External job created for %s with type %s", rq.ForJob.JobId, rq.PredefinedType)
//		pushErr := pushToRunningQueue(j.redisClient, rq)
//		if pushErr != nil {
//			log.Print("Could not update running queue! ", pushErr)
//			return pushErr
//		}
//		return nil
//	} else {
//		log.Print("Other job types not yet implemented! ", rq.PredefinedType)
//		return errors.New("other job types not yet implemented")
//	}
//}

func (j *JobRunner) actionRequest(container *models.JobContainer) error {
	initialStep := container.InitialStep()
	return j.actionStep(initialStep)
}

func (j *JobRunner) actionStep(step models.JobStep) error {
	analysisJob, isAnalysis := step.(models.JobStepAnalysis)
	if isAnalysis {
		err := CreateAnalysisJob(analysisJob, j.k8client)
		if err != nil {
			log.Print("Could not create analysis job! ", err)
			return err
		}
		log.Printf("External job created for %s with type analysis", analysisJob.JobStepId)
		pushErr := pushToRunningQueue(j.redisClient, &step)
		if pushErr != nil {
			log.Print("Could not save job to queue: ", pushErr)
			return err
		}
		return nil
	}

	_, isThumb := step.(models.JobStepThumbnail)
	if isThumb {
		log.Print("Thumbnail job not implemented yet")
		return errors.New("Thumbnail job not implemented yet")
	}

	return errors.New("Did not recognise initial step type")
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

	for _, runningJob := range *queueSnapshot {
		jobId := runningJob.StepId()

		runners, runErr := FindRunnerFor(jobId, j.k8client)
		if runErr != nil {
			log.Print("Could not get runner for ", jobId, ": ", runErr)
			continue //proceed to next one, don't abort
		}

		var updatedJob JobRunnerRequest
		copyErr := copier.Copy(&updatedJob, &runningJob)
		if copyErr != nil {
			log.Print("ERROR: Could not perform copy (out of memory?!): ", copyErr)
			continue
		}

		for _, runner := range *runners {
			switch runner.Status {
			case models.CONTAINER_COMPLETED:
				//updatedJob.ForJob.Status = models.JOB_COMPLETED
				//log.Printf("External job for %s with type %s completed", jobId, runningJob.PredefinedType)
				//putErr := models.PutJob(&updatedJob.ForJob, j.redisClient)
				//if putErr != nil {
				//	log.Print("Could not update job ", updatedJob.ForJob, ": ", putErr)
				//} else {
				//	removeFromQueue(j.redisClient, RUNNING_QUEUE, &runningJob)
				//}
				container := models.JobContainerForId(runningJob.ContainerId())
				nextStep := container.CompleteStepAndMoveOn()
				if nextStep != nil {
					j.actionStep(nextStep)
				}
			case models.CONTAINER_FAILED:
				updatedJob.ForJob.Status = models.JOB_FAILED
				log.Printf("External job for %s with type %s failed", runningJob.ForJob.JobId, runningJob.PredefinedType)
				putErr := models.PutJob(&updatedJob.ForJob, j.redisClient)
				if putErr != nil {
					log.Print("Could not update job ", updatedJob.ForJob, ": ", putErr)
				} else {
					removeFromQueue(j.redisClient, RUNNING_QUEUE, &runningJob)
				}
			case models.CONTAINER_ACTIVE:
				updatedJob.ForJob.Status = models.JOB_STARTED
				log.Printf("External job for %s with type %s is active", runningJob.ForJob.JobId, runningJob.PredefinedType)
				putErr := models.PutJob(&updatedJob.ForJob, j.redisClient)
				if putErr != nil {
					log.Print("Could not update job ", updatedJob.ForJob, ": ", putErr)
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

		newJob, getErr := getNextRequestQueueEntry(j.redisClient)
		if getErr == nil {
			if newJob == nil {
				//log.Printf("No more jobs to get")
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
