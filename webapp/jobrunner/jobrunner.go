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
	templateMgr  *models.JobTemplateManager
	maxJobs         int32
}

/**
create a new JobRunner object
*/
func NewJobRunner(redisClient *redis.Client, k8client *kubernetes.Clientset, templateManager *models.JobTemplateManager, channelBuffer int, maxJobs int32) JobRunner {
	shutdownChan := make(chan bool)
	queuePollTicker := time.NewTicker(1 * time.Second)

	runner := JobRunner{
		redisClient:     redisClient,
		k8client:        k8client,
		shutdownChan:    shutdownChan,
		queuePollTicker: queuePollTicker,
		templateMgr: templateManager,
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

	for _, jobStep := range *queueSnapshot {
		jobId := jobStep.StepId()

		runners, runErr := FindRunnerFor(jobId, j.k8client)
		if runErr != nil {
			log.Print("Could not get runner for ", jobId, ": ", runErr)
			continue //proceed to next one, don't abort
		}

		if len(*runners)>1 {
			log.Printf("WARNING: Got %d runners for jobstep ID %s, should only have one. Using the first with container id: %s", len(*runners), jobId, (*runners)[0].JobUID)
		}
		runner := (*runners)[0]
		switch runner.Status {
		case models.CONTAINER_COMPLETED:
			removeFromQueue(j.redisClient, RUNNING_QUEUE, &jobStep)

			container, getErr := models.JobContainerForId(jobStep.ContainerId(), j.redisClient)
			if getErr != nil {
				log.Printf("Could not get job master data for %s: %s", jobStep.ContainerId(), getErr)
				continue	//pick it up on the next iteration
			}
			log.Printf("External job step %s completed", jobStep.StepId())
			nextStep := container.CompleteStepAndMoveOn()	//this updates the internal state of `container`

			storErr := container.Store(j.redisClient)
			if storErr != nil {
				log.Printf("Could not store job container: %s", storErr)
			} else {
				log.Printf("Job completed and saved")
			}

			if nextStep != nil { //nil => this was the last jobstep, not nil => another step to queue
				log.Printf("Job %s: Moving to next job step ", container.Id)
				runErr := j.actionStep(nextStep)
				if runErr != nil {
					log.Print("Could not action next step: ", runErr)
					container.Status = models.JOB_FAILED
					container.ErrorMessage = runErr.Error()
					storErr = container.Store(j.redisClient)
					if storErr != nil {
						log.Printf("Could not store updated job container: %s", storErr)
					}
				}
			}
		case models.CONTAINER_FAILED:
			//updatedJob.ForJob.Status = models.JOB_FAILED
			//log.Printf("External job for %s with type %s failed", runningJob.ForJob.JobId, runningJob.PredefinedType)
			//putErr := models.PutJob(&updatedJob.ForJob, j.redisClient)
			//if putErr != nil {
			//	log.Print("Could not update job ", updatedJob.ForJob, ": ", putErr)
			//} else {
			//	removeFromQueue(j.redisClient, RUNNING_QUEUE, &runningJob)
			//}
			
		case models.CONTAINER_ACTIVE:
			updatedJob.ForJob.Status = models.JOB_STARTED
			log.Printf("External job for %s with type %s is active", runningJob.ForJob.JobId, runningJob.PredefinedType)
			putErr := models.PutJob(&updatedJob.ForJob, j.redisClient)
			if putErr != nil {
				log.Print("Could not update job ", updatedJob.ForJob, ": ", putErr)
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
