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
	Id             uuid.UUID `json:"id"`
	Steps          []JobStep `json:"steps"`
	CompletedSteps int       `json:"completed_steps"`
	Status         JobStatus `json:"status"`
	JobTemplateId  uuid.UUID `json:"templateId"`
	ErrorMessage  string `json:"error_message"`
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
	c.Steps[c.CompletedSteps] = c.Steps[c.CompletedSteps].WithNewStatus(JOB_COMPLETED)
	c.CompletedSteps += 1
	if c.CompletedSteps>= len(c.Steps) {
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
