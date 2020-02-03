package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
	"time"
)

type JobStatus int

const (
	JOB_PENDING JobStatus = iota
	JOB_STARTED
	JOB_COMPLETED
	JOB_FAILED
)

type JobSort int

const (
	SORT_NONE JobSort = iota
	SORT_CTIME
)

const (
	REDIDX_CTIME = "mediaflipper:jobcontainer:starttimeindex"
)

//containers are initiated from JobTemplateManager, so there is no New function
type JobContainer struct {
	Id                uuid.UUID  `json:"id"`
	Steps             []JobStep  `json:"steps"`
	CompletedSteps    int        `json:"completed_steps"`
	Status            JobStatus  `json:"status"`
	JobTemplateId     uuid.UUID  `json:"templateId"`
	ErrorMessage      string     `json:"error_message"`
	IncomingMediaFile string     `json:"incoming_media_file"`
	StartTime         *time.Time `json:"start_time"`
	EndTime           *time.Time `json:"end_time"`
}

func (c JobContainer) Store(redisClient *redis.Client) error {
	dbKey := fmt.Sprintf("mediaflipper:JobContainer:%s", c.Id)

	content, marshalErr := json.Marshal(c)
	if marshalErr != nil {
		log.Printf("Could not marshal data for job container %s: %s", c.Id, marshalErr)
		return marshalErr
	}

	_, saveErr := redisClient.Set(dbKey, string(content), -1).Result()
	if saveErr != nil {
		log.Printf("Could not save data for job container %s: %s", c.Id, saveErr)
		return saveErr
	}
	return nil
}

/**
return a copy of the current jobstep
*/
func (c *JobContainer) CurrentStep() JobStep {
	return c.Steps[c.CompletedSteps]
}

func (c *JobContainer) CompleteStepAndMoveOn() JobStep {
	log.Printf("completing step %d / %d", c.CompletedSteps, len(c.Steps))

	if len(c.Steps) > c.CompletedSteps {
		c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_COMPLETED, nil)
		c.CompletedSteps += 1
	} else {
		log.Printf("WARNING: data issue, completedsteps counter is larger than step list length?")
	}

	if c.CompletedSteps >= len(c.Steps) {
		c.Status = JOB_COMPLETED
		nowTime := time.Now()
		c.EndTime = &nowTime
		return nil
	}
	nextStep := c.Steps[c.CompletedSteps]
	return nextStep
}

func (c *JobContainer) InitialStep() JobStep {
	if len(c.Steps) == 0 {
		c.Status = JOB_COMPLETED
		nowTime := time.Now()
		c.EndTime = &nowTime
		return nil
	}
	return c.Steps[0]
}

func (c *JobContainer) FailCurrentStep(msg string) {
	c.Status = JOB_FAILED
	nowTime := time.Now()
	c.EndTime = &nowTime
	c.ErrorMessage = msg
	c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_FAILED, &msg)
}

/**
iterate the internal step list and try to find a step with the given ID
returns a pointer to freshly copied JobStep data if found or nil if not found.
*/
func (c JobContainer) FindStepById(stepId uuid.UUID) *JobStep {
	for _, s := range c.Steps {
		if s.StepId() == stepId {
			newStep := s //copy out the data rather than referencing
			return &newStep
		}
	}
	return nil //nothing found, return nil
}

/**
update the jobstep with the given ID to a new value
*/
func (c *JobContainer) UpdateStepById(stepId uuid.UUID, updatedStep JobStep) error {
	for i, s := range c.Steps {
		if s.StepId() == stepId {
			c.Steps[i] = updatedStep
			return nil
		}
	}
	return errors.New("No step found for that ID")
}

/**
sets the media file on the job and any job steps that need it
*/
func (c *JobContainer) SetMediaFile(newMediaFile string) {
	c.IncomingMediaFile = newMediaFile

	for i, step := range c.Steps {
		c.Steps[i] = step.WithNewMediaFile(newMediaFile)
	}
}

func (c *JobContainer) UnmarshalJSON(data []byte) error {
	var rawDataMap map[string]interface{}
	err := json.Unmarshal(data, &rawDataMap)
	if err != nil {
		return err
	}

	rawSteps := rawDataMap["steps"].([]interface{})
	steps := make([]JobStep, len(rawSteps))
	//spew.Dump(rawSteps)
	for i, untypedRawStep := range rawSteps {
		rawStep := untypedRawStep.(map[string]interface{})
		stepType, typeIsString := rawStep["stepType"].(string)
		if !typeIsString {
			return errors.New("stepType was not a string")
		}
		switch stepType {
		case "analysis":
			decodedStep, decErr := JobStepAnalysisFromMap(rawStep)
			if decErr != nil {
				log.Printf("decoding ERROR: %s for %s", decErr, spew.Sdump(rawStep))
				return decErr
			}
			steps[i] = decodedStep
		case "thumbnail":
			decodedStep, decErr := JobStepThumbnailFromMap(rawStep)
			if decErr != nil {
				log.Printf("decoding ERROR: %s for %s", decErr, spew.Sdump(rawStep))
				return decErr
			}
			steps[i] = decodedStep
		default:
			log.Printf("WARNING: Did not recognise job step type %s", rawStep["stepType"].(string))
		}
	}

	c.Steps = steps
	c.IncomingMediaFile = rawDataMap["incoming_media_file"].(string)
	c.Status = JobStatus(rawDataMap["status"].(float64))
	c.CompletedSteps = int(rawDataMap["completed_steps"].(float64))
	c.ErrorMessage = rawDataMap["error_message"].(string)
	c.Id = uuid.MustParse(rawDataMap["id"].(string))
	c.JobTemplateId = uuid.MustParse(rawDataMap["templateId"].(string))
	c.StartTime = timeFromOptionalString(rawDataMap["start_time"])
	c.EndTime = timeFromOptionalString(rawDataMap["end_time"])
	return nil
}

func JobContainerForId(forId uuid.UUID, redisClient *redis.Client) (*JobContainer, error) {
	dbKey := fmt.Sprintf("mediaflipper:JobContainer:%s", forId)

	content, getErr := redisClient.Get(dbKey).Result()
	if getErr != nil {
		log.Printf("Could not retrieve job container with id %s: %s", forId, getErr)
		return nil, getErr
	}

	var c JobContainer
	marshalErr := json.Unmarshal([]byte(content), &c)
	if marshalErr != nil {
		log.Printf("Could not unmarshal data from store: %s. Offending data was: %s", marshalErr, content)
		return nil, marshalErr
	}
	return &c, nil
}

/**
scans for data matching job containers and retrieves up to `limit` records starting from `cursor`.
returns a pointer to an array of containers (if successful), a new cursor to continue iterating (if successful)
and an error (if failed)
Note, consider switching to msgpack https://msgpack.org/index.html when moving to production

parameters:
- cursor - for SORT_NONE the iteration cursor, for SORT_CTIME the latest item  index to get. 0 for "since the top of the list" i.e. latest
- limit - for SORT_NONE the number of items to return, for SORT_CTIME the earliest item index to get. -1 for "everything".
*/
func ListJobContainersJson(cursor uint64, limit int64, redisclient *redis.Client, sort JobSort) (*[]string, uint64, error) {
	var keys []string
	var nextCursor uint64
	var scanErr error

	switch sort {
	case SORT_NONE:
		keys, nextCursor, scanErr = redisclient.Scan(cursor, "mediaflipper:JobContainer:*", limit).Result()
		break
	case SORT_CTIME:
		jobIdList, err := redisclient.ZRevRange(REDIDX_CTIME, int64(cursor), limit).Result()
		scanErr = err

		if err == nil {
			keys = make([]string, len(jobIdList))
			for i, jobId := range jobIdList {
				keys[i] = "mediaflipper:JobContainer:" + jobId
			}
		}
		break
	default:
		scanErr = errors.New("unknown JobSort value")
	}

	if scanErr != nil {
		log.Printf("Could not scan job containers: %s", scanErr)
		return nil, 0, scanErr
	}

	pipe := redisclient.Pipeline()
	defer pipe.Close()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(key)
	}

	_, getErr := pipe.Exec()
	if getErr != nil {
		log.Print("Could not retrieve job data: ", getErr)
		return nil, 0, scanErr
	}

	rtn := make([]string, len(cmds))
	for i, cmd := range cmds {
		content, _ := cmd.Result()
		rtn[i] = content
	}
	return &rtn, nextCursor, nil
}

func ListJobContainers(cursor uint64, limit int64, redisclient *redis.Client, sort JobSort) (*[]JobContainer, uint64, error) {
	jsonBlobs, nextCursor, scanErr := ListJobContainersJson(cursor, limit, redisclient, sort)
	if scanErr != nil {
		return nil, 0, scanErr
	}

	rtn := make([]JobContainer, len(*jsonBlobs))
	for i, blob := range *jsonBlobs {
		marshalErr := json.Unmarshal([]byte(blob), &rtn[i])
		if marshalErr != nil {
			log.Printf("Could not unmarshal data for entry %d: %s. Offending data was %s.", i, marshalErr, blob)
			return nil, 0, marshalErr
		}
	}
	return &rtn, nextCursor, nil
}

/**
reindexes a single page of results
parameters:
- uint64 representing the cursor to iterate from. Pass 0 on first call
- uint64 representing the number of items to get in a page
- redis.Pipeliner. Commands are added to the pipeline, it's the caller's responsibility to Exec() it
- redis.Client, client to pull results from
return:
- int representing the number of items processed
- uint64 representing the cursor to continue iterating. If non-zero this should be called repeatedly with the `cursor` parameter updated
- an error if something went wrong
*/
func indexNextPage(cursor uint64, limit int64, p redis.Pipeliner, client *redis.Client) (int, uint64, error) {
	log.Printf("INFO: indexNextPage from %d with a limit of %d", cursor, limit)
	contentPtr, nextCursor, err := ListJobContainers(cursor, limit, client, SORT_NONE)
	if err != nil {
		return 0, 0, err
	}

	for _, jobInfo := range *contentPtr {
		score := jobInfo.StartTime.UnixNano()
		p.ZAdd(REDIDX_CTIME, &redis.Z{
			Score:  float64(score),
			Member: jobInfo.Id.String(),
		})
	}
	return len(*contentPtr), nextCursor, nil
}

func ReIndexJobContainers(redisclient *redis.Client) error {
	log.Printf("Starting re-index of job containers")
	startTime := time.Now().Unix()

	log.Printf("DEBUG: Removing existing index")
	redisclient.Del(REDIDX_CTIME)

	log.Printf("DEBUG: Building new index")

	var nextCursor uint64 = 0
	processedItemsTotal := 0
	page := 1
	for {
		pipe := redisclient.Pipeline()
		processedItemsPage, cur, err := indexNextPage(nextCursor, 100, pipe, redisclient)
		nextCursor = cur
		processedItemsTotal += processedItemsPage
		if err != nil {
			log.Printf("ERROR: Could not index page %d: %s", page, err)
			return err
		}

		if processedItemsPage > 0 {
			log.Printf("Committing %d processed items...", processedItemsPage)
			_, putErr := pipe.Exec()
			if putErr != nil {
				log.Printf("Could not output index entries: %s", putErr)
				return putErr
			}
		}
		if nextCursor == 0 {
			log.Printf("Completed iterating items")
			break
		}
	}

	endTime := time.Now().Unix()
	timeTaken := endTime - startTime
	log.Printf("Reindex run of %d items completed in %d seconds", processedItemsTotal, timeTaken)
	return nil
}
