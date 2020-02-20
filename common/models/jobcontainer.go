package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"log"
	"reflect"
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
	SORT_CTIME_OLDEST
)

const (
	REDIDX_CTIME = "mediaflipper:jobcontainer:starttimeindex"
)

//containers are initiated from JobTemplateManager, so there is no New function
type JobContainer struct {
	Id                 uuid.UUID            `json:"id"`
	Steps              []JobStep            `json:"steps"`
	CompletedSteps     int                  `json:"completed_steps"`
	Status             JobStatus            `json:"status"`
	JobTemplateId      uuid.UUID            `json:"templateId"`
	ErrorMessage       string               `json:"error_message"`
	IncomingMediaFile  string               `json:"incoming_media_file"`
	StartTime          *time.Time           `json:"start_time"`
	EndTime            *time.Time           `json:"end_time"`
	AssociatedBulkItem *uuid.UUID           `json:"associated_bulk_item"`
	ItemType           helpers.BulkItemType `json:"item_type"`
	ThumbnailId        *uuid.UUID           `json:"thumbnail_id"`
	TranscodedMediaId  *uuid.UUID           `json:"transcoded_media_id"`
}

/**
remove any dependant objects from the datastore. This calls out to each job step and asks them to perform the operation.
*/
func (c JobContainer) DeleteAssociatedItems(redisClient redis.Cmdable) []error {
	errorList := make([]error, 0)

	for _, s := range c.Steps {
		newErrors := s.DeleteAssociatedItems(redisClient)
		errorList = append(errorList, newErrors...)
	}

	//we can't delete an associated bulk item here as we should do it from the whole bulk list
	return errorList
}

/**
remove this object from the datastore. You should make sure you call DeleteAssociatedItems first
to prevent items becoming orphaned in the store.
*/
func (c JobContainer) Remove(redisClient redis.Cmdable) error {
	dbKey := fmt.Sprintf("mediaflipper:JobContainer:%s", c.Id)
	_, err := redisClient.Del(dbKey).Result()
	if err == nil {
		idxErr := removeFromIndex(c.Id, redisClient)
		return idxErr
	}
	return err
}

func (c JobContainer) Store(redisClient redis.Cmdable) error {
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
	idxErr := indexSingleEntry(&c, redisClient)
	if idxErr != nil {
		log.Printf("Could not store index data for job container %s: %s", c.Id, idxErr)
		return idxErr
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
	if c.CompletedSteps >= len(c.Steps) {
		log.Printf("ERROR: Trying to fail current step when all steps have already been processed? Offending data was %s", spew.Sdump(c))
	} else {
		c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_FAILED, &msg)
	}
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

func optionalUuid(from map[string]interface{}, key string, forId string) *uuid.UUID {
	var thumbnailIdPtr *uuid.UUID
	if thumbId, haveThumbId := from[key]; haveThumbId {
		if thumbId == nil {
			thumbnailIdPtr = nil
		} else {
			thumbIdStr, isStr := thumbId.(string)
			if !isStr {
				log.Printf("ERROR: thumbnail ID for %s is not a string!", forId)
			} else {
				thumbnailUuid, uuidErr := uuid.Parse(thumbIdStr)
				if uuidErr != nil {
					log.Printf("ERROR: could not parse thumbnail id %s as a uuid: %s", thumbIdStr, uuidErr)
				} else {
					thumbnailIdPtr = &thumbnailUuid
				}
			}
		}
	}
	return thumbnailIdPtr
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
		rawStep, isMap := untypedRawStep.(map[string]interface{})
		if !isMap {
			log.Printf("ERROR: job data is not valid, had a step that is of type %s not map", reflect.TypeOf(untypedRawStep))
			log.Printf("Offending job data: %s", spew.Sdump(rawDataMap))
			continue
		}
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
		case "transcode":
			decodedStep, decErr := JobStepTranscodeFromMap(rawStep)
			if decErr != nil {
				log.Printf("decoding ERROR: %s for %s", decErr, spew.Sdump(rawStep))
				return decErr
			}
			steps[i] = decodedStep
		case "custom":
			decodedStep, decErr := JobStepCustomFromMap(rawStep)
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
	c.StartTime = TimeFromOptionalString(rawDataMap["start_time"])
	c.EndTime = TimeFromOptionalString(rawDataMap["end_time"])
	c.ThumbnailId = optionalUuid(rawDataMap, "thumbnail_id", rawDataMap["id"].(string))
	c.TranscodedMediaId = optionalUuid(rawDataMap, "transcoded_media_id", rawDataMap["id"].(string))
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
		log.Printf("DEBUG: getting revrange on %s", REDIDX_CTIME)
		jobIdList, err := redisclient.ZRevRange(REDIDX_CTIME, int64(cursor), limit).Result()
		scanErr = err
		log.Printf("DEBUG: index gave a total of %d items with a limit of %d", len(jobIdList), limit)
		if err == nil {
			keys = make([]string, len(jobIdList))
			for i, jobId := range jobIdList {
				keys[i] = "mediaflipper:JobContainer:" + jobId
			}
		}
		break
	case SORT_CTIME_OLDEST:
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

	results, _ := pipe.Exec()

	log.Printf("DEBUG: pipelining a total of %d commands", len(cmds))
	rtn := make([]string, 0)
	for _, r := range results {
		cmd := r.(*redis.StringCmd)

		content, getErr := cmd.Result()
		if getErr != nil {
			log.Printf("could not %s: %s", cmd.String(), getErr)
		}
		if content != "" {
			rtn = append(rtn, content)
		}
	}
	return &rtn, nextCursor, nil
}

func ListJobContainers(cursor uint64, limit int64, redisclient *redis.Client, sort JobSort) (*[]JobContainer, uint64, error) {
	jsonBlobs, nextCursor, scanErr := ListJobContainersJson(cursor, limit, redisclient, sort)
	if scanErr != nil {
		return nil, 0, scanErr
	}

	if jsonBlobs == nil {
		log.Printf("No jobs to list!")
		rtnList := make([]JobContainer, 0)
		return &rtnList, 0, nil
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
adds a single entry to the ctime index
*/
func indexSingleEntry(ent *JobContainer, client redis.Cmdable) error {
	log.Printf("indexing job %s", ent.Id)
	_, err := client.ZAdd(REDIDX_CTIME, &redis.Z{
		Score:  float64(ent.StartTime.UnixNano()),
		Member: ent.Id.String(),
	}).Result()
	return err
}

func removeFromIndex(forId uuid.UUID, client redis.Cmdable) error {
	log.Printf("removing job %s from index", forId)
	_, err := client.ZRem(REDIDX_CTIME, forId.String()).Result()
	return err
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
	log.Printf("DEBUG: indexNextPage queued %d index entries", len(*contentPtr))
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
