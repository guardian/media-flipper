package models

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"log"
)

type JobStatus int

const (
	JOB_PENDING JobStatus = iota
	JOB_STARTED
	JOB_COMPLETED
	JOB_FAILED
)

//containers are initiated from JobTemplateManager
type JobContainer struct {
	Id                uuid.UUID `json:"id"`
	Steps             []JobStep `json:"steps"`
	CompletedSteps    int       `json:"completed_steps"`
	Status            JobStatus `json:"status"`
	JobTemplateId     uuid.UUID `json:"templateId"`
	ErrorMessage      string    `json:"error_message"`
	IncomingMediaFile string    `json:"incoming_media_file"`
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
	c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_COMPLETED, nil)
	c.CompletedSteps += 1
	if c.CompletedSteps >= len(c.Steps) {
		c.Status = JOB_COMPLETED
		return nil
	}
	nextStep := c.Steps[c.CompletedSteps]
	return nextStep
}

func (c *JobContainer) InitialStep() JobStep {
	if len(c.Steps) == 0 {
		c.Status = JOB_COMPLETED
		return nil
	}
	return c.Steps[0]
}

func (c *JobContainer) FailCurrentStep(msg string) {
	c.Status = JOB_FAILED
	c.ErrorMessage = msg
	c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_FAILED, &msg)
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
*/
func ListJobContainersJson(cursor uint64, limit int64, redisclient *redis.Client) (*[]string, uint64, error) {
	keys, nextCursor, scanErr := redisclient.Scan(cursor, "mediaflipper:JobContainer:*", limit).Result()
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

func ListJobContainers(cursor uint64, limit int64, redisclient *redis.Client) (*[]JobContainer, uint64, error) {
	jsonBlobs, nextCursor, scanErr := ListJobContainersJson(cursor, limit, redisclient)
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
