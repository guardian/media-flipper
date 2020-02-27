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
	"strings"
	"time"
)

type JobStatus int

const (
	JOB_PENDING JobStatus = iota
	JOB_STARTED
	JOB_COMPLETED
	JOB_FAILED
	JOB_ABORTED
	JOB_NOT_QUEUED
)

type JobSort int

const (
	SORT_NONE JobSort = iota
	SORT_CTIME
	SORT_CTIME_OLDEST
)

const (
	REDIDX_CTIME               = "mediaflipper:jobcontainer:starttimeindex"
	JOBIDX_STATUS              = "mediaflipper:jobcontainer:statusindex"
	JOBIDX_BULKITEMASSOCIATION = "mediaflipper:jobcontainer:bulkassociation:item" //hash-table index. Key is the uuid of the bulk item we are associated with and value is the id of the job
)

type BulkAssociation struct {
	Item uuid.UUID `json:"item"`
	List uuid.UUID `json:"list"`
}

//containers are initiated from JobTemplateManager, so there is no New function
type JobContainer struct {
	Id                uuid.UUID            `json:"id"`
	Steps             []JobStep            `json:"steps"`
	CompletedSteps    int                  `json:"completed_steps"`
	Status            JobStatus            `json:"status"`
	JobTemplateId     uuid.UUID            `json:"templateId"`
	ErrorMessage      string               `json:"error_message"`
	IncomingMediaFile string               `json:"incoming_media_file"`
	StartTime         *time.Time           `json:"start_time"`
	EndTime           *time.Time           `json:"end_time"`
	AssociatedBulk    *BulkAssociation     `json:"associated_bulk"`
	ItemType          helpers.BulkItemType `json:"item_type"`
	ThumbnailId       *uuid.UUID           `json:"thumbnail_id"`
	TranscodedMediaId *uuid.UUID           `json:"transcoded_media_id"`
	OutputPath        string               `json:"output_path"` //optional output location
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
		idxErr := removeFromIndex(c.Id, c.AssociatedBulk, redisClient)
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
	outputPath, haveOutputPath := rawDataMap["output_path"]
	if haveOutputPath {
		c.OutputPath = outputPath.(string)
	}

	_, haveAssocBulk := rawDataMap["associated_bulk"]
	if haveAssocBulk && rawDataMap["associated_bulk"] != nil {
		associatedBulkRaw := rawDataMap["associated_bulk"].(map[string]interface{})
		associatedBulkContent := &BulkAssociation{Item: safeGetUUID(associatedBulkRaw["item"].(string)), List: safeGetUUID(associatedBulkRaw["list"].(string))}
		c.AssociatedBulk = associatedBulkContent
	}
	return nil
}

func JobContainerForId(forId uuid.UUID, redisClient redis.Cmdable) (*JobContainer, error) {
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
func ListJobContainersJson(cursor uint64, limit int64, redisclient *redis.Client, sort JobSort, maybeStatus *JobStatus) (*[]string, uint64, error) {
	var keys []string
	var nextCursor uint64
	var scanErr error

	var indexName string
	if maybeStatus == nil {
		indexName = REDIDX_CTIME
	} else {
		indexName = fmt.Sprintf("%s:%d", JOBIDX_STATUS, *maybeStatus)
	}

	switch sort {
	case SORT_NONE:
		keys, nextCursor, scanErr = redisclient.Scan(cursor, "mediaflipper:JobContainer:*", limit).Result()
		break
	case SORT_CTIME:
		log.Printf("DEBUG: getting revrange on %s", indexName)
		jobIdList, err := redisclient.ZRevRange(indexName, int64(cursor), limit).Result()
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
		jobIdList, err := redisclient.ZRevRange(indexName, int64(cursor), limit).Result()
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

func ListJobContainers(cursor uint64, limit int64, redisclient *redis.Client, sort JobSort, maybeStatus *JobStatus) (*[]JobContainer, uint64, error) {
	jsonBlobs, nextCursor, scanErr := ListJobContainersJson(cursor, limit, redisclient, sort, maybeStatus)
	if scanErr != nil {
		return nil, 0, scanErr
	}

	if jsonBlobs == nil {
		log.Printf("No jobs to list!")
		rtnList := make([]JobContainer, 0)
		return &rtnList, 0, nil
	}

	log.Printf("DEBUG: ListJobContainers top container is %s", spew.Sdump((*jsonBlobs)[0]))

	rtn := make([]JobContainer, len(*jsonBlobs))
	for i, blob := range *jsonBlobs {
		marshalErr := json.Unmarshal([]byte(blob), &rtn[i])
		if marshalErr != nil {
			log.Printf("Could not unmarshal data for entry %d: %s. Offending data was %s.", i, marshalErr, blob)
			return nil, 0, marshalErr
		}
	}
	log.Printf("DEBUG: ListJobContainers top container is %s", spew.Sdump(rtn[0]))
	return &rtn, nextCursor, nil
}

func fetchJobContainersList(idList []uuid.UUID, redisClient redis.Cmdable) ([]JobContainer, error) {
	pipe := redisClient.Pipeline()

	//cmds := make([]*redis.StringCmd, len(idList))
	for _, jobId := range idList {
		pipe.Get(fmt.Sprintf("mediaflipper:JobContainer:%s", jobId))
	}

	results, pipeErr := pipe.Exec()
	if pipeErr != nil {
		log.Printf("ERROR fetchJobContainersList could not retrieve items: %s", pipeErr)
		return nil, pipeErr
	}

	output := make([]JobContainer, len(results))

	for i, result := range results {
		cmd := result.(*redis.StringCmd)
		content, _ := cmd.Result()

		marshalErr := json.Unmarshal([]byte(content), &output[i])
		if marshalErr != nil {
			log.Printf("ERROR fetchJobContainersList could not parse content from datastore for %s: %s", cmd.String(), marshalErr)
		}
	}
	return output, nil
}

/**
get job data associated with the given bulk item.
returns:
 - nil, nil if there is no job found
 - nil, error if the retrieve fails
 - ptr to JobContainer, nil if the retrieve succeeds
*/
func JobContainerForBulkItem(bulkItemId uuid.UUID, redisClient redis.Cmdable) ([]JobContainer, error) {
	jobIdListStr, getErr := redisClient.HGet(JOBIDX_BULKITEMASSOCIATION, bulkItemId.String()).Result()
	log.Printf("JobContainerForBulkItem DEBUG result from index: %s %s", jobIdListStr, getErr)
	if getErr != nil {
		return nil, getErr
	}

	if jobIdListStr == "" {
		return nil, nil
	}

	parts := strings.Split(jobIdListStr, "|")
	idList := make([]uuid.UUID, 0)
	for _, idStr := range parts {
		if idStr == "" {
			continue //get rid of annoying error message if it's blank
		}
		uid, parseErr := uuid.Parse(idStr)
		if parseErr == nil {
			idList = append(idList, uid)
		} else {
			log.Printf("WARNING JobContainerForBulkItem could not parse value '%s' from datastore: %s", idStr, parseErr)
		}
	}

	log.Printf("JobContainerForBulkItem DEBUG got id list: %s", idList)
	return fetchJobContainersList(idList, redisClient)
}

/**
concatenate the given data to the given hashkey via a lua script on the
redis datastore
*/
func indexLuaConcat(ent *JobContainer, client redis.Cmdable) error {
	/**
	luaScript expects to be called with 3 arguments:
	- ARGV[1] - name of the index
	- ARGV[2] - id of the associated item to link to
	- ARGV[3] - id of the job to link from
	*/

	luaScript := `local currentValue = redis.call("hget",ARGV[1],ARGV[2])
if currentValue == false then
	currentValue = ""
end

local comparisonString = string.gsub(ARGV[3],"-","%%-")
if string.find(currentValue, comparisonString) then
	return currentValue
else
	local replacement = currentValue .. "|" .. ARGV[3]
	redis.call("hset",ARGV[1],ARGV[2],replacement)
	return replacement
end
`
	if ent.AssociatedBulk != nil {
		_, err := client.Eval(luaScript, []string{}, JOBIDX_BULKITEMASSOCIATION, ent.AssociatedBulk.Item.String(), ent.Id.String()).Result()
		return err
	}
	return nil
}

/**
remove the given data from the given hashkey via a lua script on the
redis datastore
*/
func indexLuaRemove(jobId uuid.UUID, bulk *BulkAssociation, client redis.Cmdable) error {
	/**
	luaScript expects to be called with 3 arguments:
	- ARGV[1] - name of the index
	- ARGV[2] - id of the associated item to link to
	- ARGV[3] - id of the job that we are removing link from
	*/

	luaScript := `local currentValue = redis.call("hget",ARGV[1],ARGV[2])
local final = {}
for value in string.gmatch(currentValue, "([^|]+)") do
	if value ~= ARGV[3] then
		table.insert(final, value)
	end
end
local replacement = table.concat(final, "|")
if replacement == "" then
	redis.call("hdel",ARGV[1],ARGV[2])
else
	redis.call("hset",ARGV[1],ARGV[2],replacement)
end
return replacement
`
	if bulk != nil {
		_, err := client.Eval(luaScript, []string{}, JOBIDX_BULKITEMASSOCIATION, bulk.Item.String(), jobId.String()).Result()
		return err
	}
	return nil
}

/**
adds a single entry to the ctime, status and associated item indices
*/
func indexSingleEntry(ent *JobContainer, client redis.Cmdable) error {
	log.Printf("indexing job %s", ent.Id)
	p := client.Pipeline()

	p.ZAdd(REDIDX_CTIME, &redis.Z{
		Score:  float64(ent.StartTime.UnixNano()),
		Member: ent.Id.String(),
	})

	statusKey := fmt.Sprintf("%s:%d", JOBIDX_STATUS, ent.Status)
	p.ZAdd(statusKey, &redis.Z{
		Score:  float64(ent.StartTime.UnixNano()),
		Member: ent.Id.String(),
	})

	indexLuaConcat(ent, p) //no point checking error as we don't execute until p.Exec()
	//if ent.AssociatedBulk != nil {
	//	p.HSet(JOBIDX_BULKITEMASSOCIATION, ent.AssociatedBulk.Item.String(), ent.Id.String())
	//}
	_, err := p.Exec()
	return err
}

func removeFromIndex(forId uuid.UUID, bulkAssociation *BulkAssociation, client redis.Cmdable) error {
	log.Printf("removing job %s from index", forId)
	p := client.Pipeline()

	p.ZRem(REDIDX_CTIME, forId.String())
	p.ZRem(JOBIDX_STATUS, forId.String())
	if bulkAssociation != nil {
		//p.HDel(JOBIDX_BULKITEMASSOCIATION, bulkAssociation.Item.String())
		indexLuaRemove(forId, bulkAssociation, client)
	}
	_, err := p.Exec()
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
	contentPtr, nextCursor, err := ListJobContainers(cursor, limit, client, SORT_NONE, nil)
	if err != nil {
		return 0, 0, err
	}

	for _, jobInfo := range *contentPtr {
		score := jobInfo.StartTime.UnixNano()
		p.ZAdd(REDIDX_CTIME, &redis.Z{
			Score:  float64(score),
			Member: jobInfo.Id.String(),
		})
		statusKey := fmt.Sprintf("%s:%d", JOBIDX_STATUS, jobInfo.Status)
		p.ZAdd(statusKey, &redis.Z{
			Score:  float64(score),
			Member: jobInfo.Id.String(),
		})
		if jobInfo.AssociatedBulk != nil {
			//p.HSet(JOBIDX_BULKITEMASSOCIATION, jobInfo.AssociatedBulk.Item.String(), jobInfo.Id.String())
			indexLuaConcat(&jobInfo, p)
		}
	}
	log.Printf("DEBUG: indexNextPage queued %d index entries", len(*contentPtr))
	return len(*contentPtr), nextCursor, nil
}

func ReIndexJobContainers(redisclient *redis.Client) error {
	log.Printf("Starting re-index of job containers")
	startTime := time.Now().Unix()

	log.Printf("DEBUG: Removing existing indices")
	redisclient.Del(REDIDX_CTIME)
	redisclient.Del(JOBIDX_BULKITEMASSOCIATION)

	log.Printf("DEBUG: Building new indices")

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
