package jobrunner

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/guardian/mediaflipper/common/models"
	"github.com/guardian/mediaflipper/webapp/bulkprocessor"
	"k8s.io/client-go/kubernetes"
	v1batch "k8s.io/client-go/kubernetes/typed/batch/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v13 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"reflect"
	"time"
)

type JobRunnerIF interface {
	AddJob(container *models.JobContainer) error
	//EnqueueContentsAsync(redisClient redis.Cmdable, templateManager models.TemplateManagerIF, l *bulkprocessor.BulkListImpl, testRunner JobRunnerIF) chan error
	clearCompletedTick()
}

type JobRunner struct {
	redisClient *redis.Client
	//k8client        *kubernetes.Clientset
	jobClient       v1batch.JobInterface
	podClient       v1.PodInterface
	serviceClient   v13.ServiceInterface
	shutdownChan    chan bool
	queuePollTicker *time.Ticker
	templateMgr     *models.JobTemplateManager
	maxJobs         int32
	bulkListDAO     bulkprocessor.BulkListDAO
}

/**
create a new JobRunner object
*/
func NewJobRunner(redisClient *redis.Client, k8client *kubernetes.Clientset, templateManager *models.JobTemplateManager, maxJobs int32, runProcessor bool) JobRunner {
	shutdownChan := make(chan bool)
	queuePollTicker := time.NewTicker(1 * time.Second)

	ns, getNsErr := GetMyNamespace()
	jobClient := k8client.BatchV1().Jobs(ns)
	serviceClient := k8client.CoreV1().Services(ns)
	podClient := k8client.CoreV1().Pods(ns)

	if getNsErr != nil {
		log.Printf("ERROR NewJobRunner could not determine current k8s namespace: %s", getNsErr)
		panic("could not determine namespace")
	}

	runner := JobRunner{
		redisClient: redisClient,
		//k8client:        k8client,
		jobClient:       jobClient,
		serviceClient:   serviceClient,
		podClient:       podClient,
		shutdownChan:    shutdownChan,
		queuePollTicker: queuePollTicker,
		templateMgr:     templateManager,
		maxJobs:         maxJobs,
		bulkListDAO:     bulkprocessor.BulkListDAOImpl{},
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
	association := container.AssociatedBulk
	if association != nil {
		updateErr := j.bulkListDAO.UpdateById(association.List, association.Item, bulkprocessor.ITEM_STATE_ACTIVE, j.redisClient)
		if updateErr != nil {
			log.Printf("ERROR: actionRequest could not update bulk state for %s: %s", association.List, updateErr)
		}
	}
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
		err := CreateAnalysisJob(*analysisJob, container.OutputPath, j.jobClient, j.serviceClient)
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
		err := CreateThumbnailJob(*thumbJob, container.OutputPath, j.jobClient, j.serviceClient)
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
		err := CreateTranscodeJob(*tcJob, container.OutputPath, j.jobClient, j.serviceClient)
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
		err := CreateCustomJob(*custJob, container, j.jobClient, j.serviceClient, j.redisClient)
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
		runners, runErr := FindRunnerFor(queueEntry.StepId, j.jobClient)
		if runErr != nil { //could not retrieve a runner from k8. Assume that this is a transient error, don't dump it from the queue
			log.Print("Could not get runner for ", queueEntry.StepId, ": ", runErr)
			removeErr := models.RemoveFromQueue(j.redisClient, models.RUNNING_QUEUE, queueEntry)
			if removeErr != nil {
				log.Printf("WARNING: Could not remove inaccurate record from running queue")
			}
			continue //proceed to next one, don't abort
		}

		if len(*runners) > 1 {
			log.Printf("WARNING clearCompletedTick Got %d runners for jobstep ID %s, should only have one. Using the first with container id: %s", len(*runners), queueEntry.StepId, (*runners)[0].JobUID)
		}
		if len(*runners) == 0 { //no runner was found. This shouldn't happen but does sometimes; handle it by assuming the job is lost and rescheduling
			log.Print("ERROR clearCompletedTick no k8 runner was found for ", queueEntry.StepId)
			container, getErr := models.JobContainerForId(queueEntry.JobId, j.redisClient)
			if getErr != nil {
				log.Printf("ERROR clearCompletedTick could not get job master data for %s: %s", queueEntry.JobId, getErr)
				continue //pick it up on the next iteration
			}
			jobStep := container.FindStepById(queueEntry.StepId)
			if jobStep == nil {
				log.Printf("ERROR clearCompletedTick job entry %s does not have a step with id %s so can't mark as lost", queueEntry.JobId, queueEntry.StepId)
			} else {
				errMsg := fmt.Sprintf("could not get any runners for step id %s", queueEntry.StepId)
				updatedStep := (*jobStep).WithNewStatus(models.JOB_LOST, &errMsg)
				updateErr := container.UpdateStepById(updatedStep.StepId(), updatedStep)
				if updateErr != nil {
					log.Printf("ERROR clearCompletedTick could not save updated job step: %s", updateErr)
				}
				container.Status = models.JOB_LOST
				storErr := container.Store(j.redisClient)
				if storErr != nil {
					log.Printf("ERROR clearcompletedTick could not store updated job: %s", storErr)
				}
			}
			removeErr := models.RemoveFromQueue(j.redisClient, models.RUNNING_QUEUE, queueEntry)
			if removeErr != nil {
				log.Printf("WARNING clearCompletedTick could not remove inaccurate record from running queue")
			}
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
				log.Printf("ERROR clearCompletedTick Could not remove jobstep from running queue: %s", removeErr)
			}

			container, getErr := models.JobContainerForId(queueEntry.JobId, j.redisClient)
			if getErr != nil {
				log.Printf("ERROR clearCompletedTick Could not get job master data for %s: %s", queueEntry.JobId, getErr)
				continue //pick it up on the next iteration
			}
			log.Printf("DEBUG clearCompletedTick External job step %s completed", queueEntry.StepId)
			nextStep := container.CompleteStepAndMoveOn() //this updates the internal state of `container`

			storErr := container.Store(j.redisClient)
			if storErr != nil {
				log.Printf("ERROR clearCompletedTick Could not store job container: %s", storErr)
			} else {
				log.Printf("DEBUG clearCompletedTick Job completed and saved")
			}

			//clean up the job and pod and extract the log, asynchronously
			go func() {
				jobStep := container.FindStepById(queueEntry.StepId)
				if jobStep == nil {
					log.Printf("WARNING clearCompletedTick could not find jobstep with ID %s within job container with id %s", queueEntry.StepId, container.Id)
				} else {
					cleanupErr := CleanUpJobStep(jobStep, j.jobClient, j.podClient, j.redisClient)
					if cleanupErr != nil {
						log.Printf("ERROR clearCompletedTick could not clean up jobstep %s for %s: %s", (*jobStep).StepId(), container.Id, cleanupErr)
					}
				}
			}()

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
			} else {
				association := container.AssociatedBulk
				if association != nil {
					log.Printf("DEBUG clearCompletedTick: updating bulk item %s in list %s to completed", association.Item, association.List)
					updateErr := j.bulkListDAO.UpdateById(association.List, association.Item, bulkprocessor.ITEM_STATE_COMPLETED, j.redisClient)
					if updateErr != nil {
						log.Printf("ERROR: actionRequest could not update bulk state for %s: %s", association.List, updateErr)
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

			association := container.AssociatedBulk
			if association != nil {
				log.Printf("DEBUG clearCompletedTick: updating bulk item %s in list %s to failed", association.Item, association.List)
				updateErr := j.bulkListDAO.UpdateById(association.List, association.Item, bulkprocessor.ITEM_STATE_FAILED, j.redisClient)
				if updateErr != nil {
					log.Printf("ERROR: actionRequest could not update bulk state for %s: %s", association.List, updateErr)
				}
			} else {
				log.Printf("DEBUG clearCompletedTick: job %s has no associated bulk item", container.Id)
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
