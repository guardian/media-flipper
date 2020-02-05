package jobrunner

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/webapp/models"
	_ "github.com/jinzhu/copier"
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
func NewJobRunner(redisClient *redis.Client, k8client *kubernetes.Clientset, templateManager *models.JobTemplateManager, channelBuffer int, maxJobs int32) JobRunner {
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

/**
trigger the action for a given item and put it onto the running queue if successful
*/
func (j *JobRunner) actionRequest(container *models.JobContainer) error {
	initialStep := container.InitialStep()
	//log.Printf("actionRequest: initialStep is %s", spew.Sdump(initialStep))
	return j.actionStep(initialStep)
}

func (j *JobRunner) actionStep(step models.JobStep) error {
	analysisJob, isAnalysis := step.(*models.JobStepAnalysis)
	if isAnalysis {
		err := CreateAnalysisJob(*analysisJob, j.k8client)
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

	thumbJob, isThumb := step.(*models.JobStepThumbnail)
	if isThumb {
		err := CreateThumbnailJob(*thumbJob, j.k8client)
		if err != nil {
			log.Print("Could not create thumbnail job! ", err)
			return err
		}
		log.Printf("External job created for %s with type thumbnail", thumbJob.JobStepId)
		pushErr := pushToRunningQueue(j.redisClient, &step)
		if pushErr != nil {
			log.Print("Could not save job to queue: ", pushErr)
			return err
		}
		return nil
	}

	tcJob, isTc := step.(*models.JobStepTranscode)
	if isTc {
		err := CreateTranscodeJob(*tcJob, j.k8client)
		if err != nil {
			log.Print("Could not create transcode job! ", err)
			return err
		}
		log.Printf("External job created for %s with type transcode", tcJob.JobStepId)
		pushErr := pushToRunningQueue(j.redisClient, &step)
		if pushErr != nil {
			log.Print("Could not save job to queue: ", pushErr)
			return err
		}
		return nil
	}

	return errors.New(fmt.Sprintf("Did not recognise step type %s", reflect.TypeOf(step)))
}

func (j *JobRunner) clearCompletedTick() {
	jobClient, clientGetErr := GetJobClient(j.k8client)
	if clientGetErr != nil {
		log.Printf("Can't clear jobs as not able to access cluster")
		return
	}

	set, checkErr := CheckQueueLock(j.redisClient, RUNNING_QUEUE)
	if checkErr != nil {
		log.Printf("Could not check running queue lock: %s", checkErr)
		return
	}

	if set {
		log.Printf("Running queue is locked, not performing clear completed")
		return
	}

	SetQueueLock(j.redisClient, RUNNING_QUEUE)

	queueSnapshot, snapErr := copyRunningQueueContent(j.redisClient)
	if snapErr != nil {
		log.Printf("ERROR: Could not clear completed jobs, queue snapshot gave an error")
		return
	}

	defer ReleaseQueueLock(j.redisClient, RUNNING_QUEUE)

	for _, jobStep := range *queueSnapshot {
		jobId := jobStep.StepId()

		runners, runErr := FindRunnerFor(jobId, jobClient)
		if runErr != nil {
			log.Print("Could not get runner for ", jobId, ": ", runErr)
			continue //proceed to next one, don't abort
		}

		if len(*runners) > 1 {
			log.Printf("WARNING: Got %d runners for jobstep ID %s, should only have one. Using the first with container id: %s", len(*runners), jobId, (*runners)[0].JobUID)
		}
		if len(*runners) == 0 {
			log.Print("Could not get runner for ", jobId, ": ", runErr)
			continue //proceed to next one, don't abort
		}
		runner := (*runners)[0]
		switch runner.Status {
		case models.CONTAINER_COMPLETED:
			/*
				remove the given step from the RUNNING_QUEUE and set its status to complete. Action the next step if there is one
				or if not complete the job and save.
			*/
			removeErr := removeFromQueue(j.redisClient, RUNNING_QUEUE, &jobStep)

			if removeErr != nil {
				log.Printf("Could not remove jobstep from running queue: %s", removeErr)
			}

			container, getErr := models.JobContainerForId(jobStep.ContainerId(), j.redisClient)
			if getErr != nil {
				log.Printf("Could not get job master data for %s: %s", jobStep.ContainerId(), getErr)
				continue //pick it up on the next iteration
			}
			log.Printf("External job step %s completed", jobStep.StepId())
			nextStep := container.CompleteStepAndMoveOn() //this updates the internal state of `container`

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
			removeFromQueue(j.redisClient, RUNNING_QUEUE, &jobStep)

			log.Printf("External job step %s failed", jobStep.StepId())

			container, getErr := models.JobContainerForId(jobStep.ContainerId(), j.redisClient)
			if getErr != nil {
				log.Printf("Could not get job master data for %s: %s", jobStep.ContainerId(), getErr)
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
			if jobStep.Status() != models.JOB_STARTED {
				container, getErr := models.JobContainerForId(jobStep.ContainerId(), j.redisClient)
				if getErr != nil {
					log.Printf("Could not get job master data for %s: %s", jobStep.ContainerId(), getErr)
					continue //pick it up on the next iteration
				}
				updatedJobStep := jobStep.WithNewStatus(models.JOB_STARTED, nil)
				//it's necessary to remove and re-add, beccause list removal in redis is by-value. if the value changes=> we lose it and go out-of-sync with the main model.
				removeFromQueue(j.redisClient, RUNNING_QUEUE, &jobStep)
				pushToRunningQueue(j.redisClient, &updatedJobStep)

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
	set, checkErr := CheckQueueLock(j.redisClient, RUNNING_QUEUE)
	if checkErr != nil {
		log.Printf("Could not check running queue lock: %s", checkErr)
		return
	}

	if set {
		log.Printf("Running queue is locked, not performing waiting queue chek")
		return
	}

	SetQueueLock(j.redisClient, RUNNING_QUEUE)

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

	ReleaseQueueLock(j.redisClient, RUNNING_QUEUE)
}
