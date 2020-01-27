package jobrunner

import (
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/webapp/models"
	"reflect"
	"testing"
)

/**
CopyRunningQueueContent should return a snapshot of the running queue as a list of models.JobStep objects
*/
func TestCopyRunningQueueContent(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	jobStepId := uuid.New()
	testStep1 := models.JobStepAnalysis{
		JobStepType:            "analysis",
		JobStepId:              jobStepId,
		JobContainerId:         uuid.New(),
		ContainerData:          nil,
		StatusValue:            models.JOB_STARTED,
		Result:                 models.AnalysisResult{},
		MediaFile:              "path/to/something",
		KubernetesTemplateFile: "mytemplate.yaml",
	}

	testStep1Enc, _ := json.Marshal(testStep1)
	testStep2 := models.JobStepThumbnail{
		JobStepType: "thumbnail",
	}
	testStep2Enc, _ := json.Marshal(testStep2)

	keyName := fmt.Sprintf("mediaflipper:%s", RUNNING_QUEUE)
	s.Lpush(keyName, string(testStep2Enc))
	s.Lpush(keyName, string(testStep1Enc))

	result, err := copyQueueContent(testClient, RUNNING_QUEUE)
	if err != nil {
		t.Error("copyQueueContent unexpectedly failed: ", err)
		t.FailNow()
	}

	if len(*result) != 2 {
		t.Errorf("expected 2 items from queue, got %d", len(*result))
		t.FailNow()
	}
	decoded := make([]models.JobStep, len(*result))
	for i, jsonBlob := range *result {
		var rawData map[string]interface{}
		json.Unmarshal([]byte(jsonBlob), &rawData)
		//spew.Dump(rawData)
		var getErr error
		decoded[i], getErr = getJobFromMap(rawData)
		if getErr != nil {
			t.Error("getJobFromMap unexpectedly failed: ", getErr)
		}
	}

	_, isAnalysis := decoded[0].(*models.JobStepAnalysis)
	if !isAnalysis {
		t.Error("Step 0 was not analysis, got ", reflect.TypeOf(decoded[0]))
	}

	_, isThumb := decoded[1].(*models.JobStepThumbnail)
	if !isThumb {
		t.Error("Step 1 was not thumbnail, got ", reflect.TypeOf(decoded[0]))
	}
}
