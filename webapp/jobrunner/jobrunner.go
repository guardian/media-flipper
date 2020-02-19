package jobrunner

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"k8s.io/client-go/kubernetes"
	"log"
	"reflect"
	"time"
)

type JobRunner struct {
	redisClient     *redis.Client
	k8client        *kubernetes.Clientset
	shutdownChan    chan bool
	queuePollTicker *time.Ticker
	templateMgr     *models.JobTemplateManager
	maxJobs         int32
}

/**
create a new JobRunner object
*/
func NewJobRunner(redisClient *redis.Client, k8client *kubernetes.Clientset, templateManager *models.JobTemplateManager, channelBuffer int, maxJobs int32, runProcessor bool) JobRunner {
	shutdownChan := make(chan bool)
	queuePollTicker := time.NewTicker(1 * time.Second)

	runner := JobRunner{
		redisClient:     redisClient,
		k8client:        k8client,
		shutdownChan:    shutdownChan,
		queuePollTicker: queuePollTicker,
		templateMgr:     templateManager,
		maxJobs:         maxJobs,
	}
	if runProcessor {
		go runner.requestProcessor()
	}
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

/**
trigger the action for a given item and put it onto the running queue if successful
*/
func (j *JobRunner) actionRequest(container *models.JobContainer) error {
	initialStep := container.InitialStep()
	if initialStep == nil {
		log.Printf("WARNING: Job %s from template %s had no steps!", container.Id.String(), container.JobTemplateId.String())
		return errors.New("job had no steps!")
	} else {
		return j.actionStep(initialStep, container)
	}
}

func (j *JobRunner) actionStep(step models.JobStep, container *models.JobContainer) error {
	analysisJob, isAnalysis := step.(*models.JobStepAnalysis)
	var newQueueEntry *models.JobQueueEntry

	if isAnalysis {
		err := CreateAnalysisJob(*analysisJob, j.k8client)
		if err != nil {
			log.Print("Could not create analysis job! ", err)
			return err
		}
		log.Printf("External job created for %s with type analysis", analysisJob.JobStepId)
		newQueueEntry = &models.JobQueueEntry{
			JobId:  step.ContainerId(),
			StepId: step.StepId(),
			Status: models.JOB_PENDING,
		}
	}

	thumbJob, isThumb := step.(*models.JobStepThumbnail)
	if isThumb {
		err := CreateThumbnailJob(*thumbJob, j.k8client)
		if err != nil {
			log.Print("Could not create thumbnail job! ", err)
			return err
		}
		log.Printf("External job created for %s with type thumbnail", thumbJob.JobStepId)
		newQueueEntry = &models.JobQueueEntry{
			JobId:  step.ContainerId(),
			StepId: step.StepId(),
			Status: models.JOB_PENDING,
		}
	}

	tcJob, isTc := step.(*models.JobStepTranscode)
	if isTc {
		err := CreateTranscodeJob(*tcJob, j.k8client)
		if err != nil {
			log.Print("Could not create transcode job! ", err)
			return err
		}
		log.Printf("External job created for %s with type transcode", tcJob.JobStepId)
		newQueueEntry = &models.JobQueueEntry{
			JobId:  step.ContainerId(),
			StepId: step.StepId(),
			Status: models.JOB_PENDING,
		}
	}

	custJob, isCust := step.(*models.JobStepCustom)
	if isCust {
		err := CreateCustomJob(*custJob, container, j.k8client, j.redisClient)
		if err != nil {
			log.Print("Could not create custom job! ", err)
			return err
		}
		log.Printf("External job created for %s with type custom", custJob.JobStepId)
		newQueueEntry = &models.JobQueueEntry{
			JobId:  step.ContainerId(),
			StepId: step.StepId(),
			Status: models.JOB_PENDING,
		}
	}

	if newQueueEntry != nil {
		pushErr := models.AddToQueue(j.redisClient, models.RUNNING_QUEUE, *newQueueEntry)
		if pushErr != nil {
			log.Printf("ERROR: Could not add to running queue: %s", pushErr)
			return pushErr
		}
		return nil
	} else {
		return errors.New(fmt.Sprintf("Did not recognise step type %s", reflect.TypeOf(step)))
	}
}

func (j *JobRunner) clearCompletedTick() {
	jobClient, clientGetErr := GetJobClient(j.k8client)
	if clientGetErr != nil {
		log.Printf("Can't clear jobs as not able to access cluster")
		return
	}

	set, checkErr := models.CheckQueueLock(j.redisClient, models.RUNNING_QUEUE)
	if checkErr != nil {
		log.Printf("Could not check running queue lock: %s", checkErr)
		return
	}

	if set {
		log.Printf("Running queue is locked, not performing clear completed")
		return
	}

	models.SetQueueLock(j.redisClient, models.RUNNING_QUEUE)

	queueSnapshot, snapErr := models.SnapshotQueue(j.redisClient, models.RUNNING_QUEUE)
	if snapErr != nil {
		log.Printf("ERROR: Could not clear completed jobs, queue snapshot gave an error")
		return
	}

	defer models.ReleaseQueueLock(j.redisClient, models.RUNNING_QUEUE) //ensure that the lock is always release!

	for _, queueEntry := range queueSnapshot {
		runners, runErr := FindRunnerFor(queueEntry.StepId, jobClient)
		if runErr != nil {
			log.Print("Could not get runner for ", queueEntry.StepId, ": ", runErr)
			continue //proceed to next one, don't abort
		}

		if len(*runners) > 1 {
			log.Printf("WARNING: Got %d runners for jobstep ID %s, should only have one. Using the first with container id: %s", len(*runners), queueEntry.StepId, (*runners)[0].JobUID)
		}
		if len(*runners) == 0 {
			log.Print("Could not get runner for ", queueEntry.StepId, ": ", runErr)
			continue //proceed to next one, don't abort
		}
		runner := (*runners)[0]
		switch runner.Status {
		case models.CONTAINER_COMPLETED:
			/*
				remove the given step from the RUNNING_QUEUE and set its status to complete. Action the next step if there is one
				or if not complete the job and save.
			*/
			removeErr := models.RemoveFromQueue(j.redisClient, models.RUNNING_QUEUE, queueEntry)

			if removeErr != nil {
				log.Printf("Could not remove jobstep from running queue: %s", removeErr)
			}

			container, getErr := models.JobContainerForId(queueEntry.JobId, j.redisClient)
			if getErr != nil {
				log.Printf("Could not get job master data for %s: %s", queueEntry.JobId, getErr)
				continue //pick it up on the next iteration
			}
			log.Printf("External job step %s completed", queueEntry.StepId)
			nextStep := container.CompleteStepAndMoveOn() //this updates the internal state of `container`

			storErr := container.Store(j.redisClient)
			if storErr != nil {
				log.Printf("Could not store job container: %s", storErr)
			} else {
				log.Printf("Job completed and saved")
			}

			if nextStep != nil { //nil => this was the last jobstep, not nil => another step to queue
				log.Printf("Job %s: Moving to next job step ", container.Id)
				runErr := j.actionStep(nextStep, container)
				if runErr != nil {
					log.Print("Could not action next step: ", runErr)
					container.Status = models.JOB_FAILED
					container.ErrorMessage = runErr.Error()
					t := time.Now()
					container.EndTime = &t
					storErr = container.Store(j.redisClient)
					if storErr != nil {
						log.Printf("Could not store updated job container: %s", storErr)
					}
				}
			}
		case models.CONTAINER_FAILED:
			/*
				remove the given step from the RUNNING_QUEUE, set the job and step status to FAILED and save
			*/
			models.RemoveFromQueue(j.redisClient, models.RUNNING_QUEUE, queueEntry)

			log.Printf("External job step %s failed", queueEntry.StepId)

			container, getErr := models.JobContainerForId(queueEntry.JobId, j.redisClient)
			if getErr != nil {
				log.Printf("Could not get job master data for %s: %s", queueEntry.JobId, getErr)
				continue //pick it up on the next iteration
			}
			container.FailCurrentStep("Kubernetes container failed")
			storErr := container.Store(j.redisClient)
			if storErr != nil {
				log.Printf("Could not store job container: %s", storErr)
			} else {
				log.Printf("Job failed and saved")
			}

		case models.CONTAINER_ACTIVE:
			/*
				check the state of the current job step. If it's not STARTED, then update it and the container statuses
				and save
			*/

			if queueEntry.Status != models.JOB_STARTED {
				container, getErr := models.JobContainerForId(queueEntry.JobId, j.redisClient)
				if getErr != nil {
					log.Printf("Could not get job master data for %s: %s", queueEntry.JobId, getErr)
					continue //pick it up on the next iteration
				}
				jobStep := container.FindStepById(queueEntry.StepId)

				updatedJobStep := (*jobStep).WithNewStatus(models.JOB_STARTED, nil)
				//it's necessary to remove and re-add, because list removal in redis is by-value. if the value changes=> we lose it and go out-of-sync with the main model.
				pipe := j.redisClient.Pipeline()
				models.RemoveFromQueue(pipe, models.RUNNING_QUEUE, queueEntry) //error handling is done in pipe.Exec()
				queueEntry.Status = models.JOB_STARTED
				models.AddToQueue(pipe, models.RUNNING_QUEUE, queueEntry)
				_, execErr := pipe.Exec()
				if execErr != nil {
					log.Printf("ERROR: could not update running queues: %s", execErr)
					return
				}

				container.Steps[container.CompletedSteps] = updatedJobStep

				if container.Status != models.JOB_STARTED {
					container.Status = models.JOB_STARTED
					t := time.Now()
					container.StartTime = &t
				}

				storErr := container.Store(j.redisClient)
				if storErr != nil {
					log.Printf("Could not store job container: %s", storErr)
				} else {
					log.Printf("Job started, container saved")
				}
			}
		}
	}
}

/**
internal function to process items on the waiting queue, up until we either run out of items on the queue or have the
max running jobs
*/
func (j *JobRunner) waitingQueueTick() {
	set, checkErr := models.CheckQueueLock(j.redisClient, models.RUNNING_QUEUE)
	if checkErr != nil {
		log.Printf("Could not check running queue lock: %s", checkErr)
		return
	}

	if set {
		log.Printf("Running queue is locked, not performing waiting queue chek")
		return
	}

	models.SetQueueLock(j.redisClient, models.RUNNING_QUEUE)
	defer models.ReleaseQueueLock(j.redisClient, models.RUNNING_QUEUE) //ensure that the lock is always released!

	for {
		//need to update and check this every iteration as we are putting stuff onto the queue
		queuelen, getErr := models.GetQueueLength(j.redisClient, models.RUNNING_QUEUE)
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
				break
			} else {
				actioningErr := j.actionRequest(newJob)
				if actioningErr != nil {
					log.Printf("Could not action job: %s", actioningErr)
					newJob.Status = models.JOB_FAILED
					newJob.ErrorMessage = actioningErr.Error()
					storeErr := newJob.Store(j.redisClient)
					if storeErr != nil {
						log.Printf("Could not save job description: %s", storeErr)
						return
					}
				} else {
					t := time.Now()
					newJob.StartTime = &t
					newJob.Store(j.redisClient)
				}
			}
		} else {
			log.Printf("Could not get next job to process!")
			break
		}
	}

}
