package analysis

import (
	"github.com/alicebob/miniredis"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	models2 "github.com/guardian/mediaflipper/common/models"
	"net/http"
	"strings"
	"testing"
	"time"
)

/*
ServeHttp should store the provided data in a new record, return the id and store it against the jobstep
*/
func TestReceiveData_ServeHTTP(t *testing.T) {
	mockRequestBody := []byte(`{"successful":true,"format":{"nb_streams":1, "nb_programs":1, "format_name": "test", "format_long_name": "test format name", "duration":12.345}}`)
	mockBody := helpers.NewMockReadCloser()
	mockBody.DataToRead = mockRequestBody

	mockRequest := http.Request{
		Method:     "POST",
		RequestURI: "https://myserver.com/api/analysis/result?forJob=E6D1337A-6850-4C15-8938-18907B2FF311&stepId=815206e7-3c09-4e0f-ad87-3a4d67767315",
		Proto:      "https",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body:       mockBody,
	}

	jobMasterId := uuid.MustParse("E6D1337A-6850-4C15-8938-18907B2FF311")
	jobStepId := uuid.MustParse("815206e7-3c09-4e0f-ad87-3a4d67767315")

	startTime := time.Now()

	fakeJobContainer := models2.JobContainer{
		Id: jobMasterId,
		Steps: []models2.JobStep{
			models2.JobStepAnalysis{
				JobStepType: "analysis",
				JobStepId:   jobStepId,
			},
		},
		CompletedSteps:    0,
		Status:            1,
		JobTemplateId:     uuid.New(),
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         &startTime,
	}

	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	defer func() {
		testClient.Close()
		s.Close()
	}()

	fakeJobContainer.Store(testClient)

	toTest := ReceiveData{redisClient: testClient}
	mockWriter := helpers.NewMockResponseWriter()
	toTest.ServeHTTP(mockWriter, &mockRequest)

	if len(mockWriter.State.LastWrittenBytes) == 0 {
		t.Error("Nothing was written to client")
	}

	if mockWriter.State.WrittenStatusCode == nil {
		t.Error("No status code was written")
	} else {
		if *mockWriter.State.WrittenStatusCode != 200 {
			t.Errorf("Incorrect status code was written, expected 200 got %d", *mockWriter.State.WrittenStatusCode)
		}
	}

	jsonContent, jsonErr := mockWriter.LastWrittenJson()
	if jsonErr != nil {
		t.Error("Method did not output json: ", jsonErr)
	} else {
		if jsonContent["status"].(string) != "ok" {
			t.Errorf("Expected status field to be 'ok', got %s", jsonContent["status"].(string))
		}
		uuidValue := jsonContent["entryId"].(string)
		parsedUuidValue, parseErr := uuid.Parse(uuidValue)
		if parseErr != nil {
			t.Errorf("returned EntryId was not a valid UUID: %s", parseErr)
		} else {
			fileFormatData, getErr := models2.GetFileFormat(parsedUuidValue, testClient)
			if getErr != nil {
				t.Error("could not retrieve file format information from the datastore")
			} else {
				if fileFormatData.FormatAnalysis.StreamCount != 1 {
					t.Error("saved format information has incorrect stream count")
				}
				if fileFormatData.FormatAnalysis.ProgCount != 1 {
					t.Error("saved format information has incorrect program count")
				}
				if fileFormatData.FormatAnalysis.FormatName != "test" {
					t.Error("saved format information has incorrect format name")
				}
				if fileFormatData.FormatAnalysis.FormatLongName != "test format name" {
					t.Error("saved format information has incorrect long format name")
				}
				if fileFormatData.FormatAnalysis.Duration != 12.345 {
					t.Error("saved format information has incorrect duration")
				}
			}

			updatedJobContainer, getErr := models2.JobContainerForId(jobMasterId, testClient)
			if getErr != nil {
				t.Error("Could not retrieve saved job")
			} else {
				updatedStep := updatedJobContainer.FindStepById(jobStepId)
				if updatedStep != nil {
					analysisStep := (*updatedStep).(*models2.JobStepAnalysis)
					if analysisStep.ResultId != parsedUuidValue {
						t.Errorf("saved job step had incorrect value, expected %s got %s", parsedUuidValue, analysisStep.ResultId)
					}
				} else {
					t.Error("Saved job did not have a step with the required id")
				}
			}
		}
		spew.Dump(jsonContent)
	}
}

/*
ServeHttp should return a 404 error if the identified job step is not an analysis step
*/
func TestReceiveData_ServeHTTP_NoCorrectSteps(t *testing.T) {
	mockRequestBody := []byte(`{"successful":true,"format":{"nb_streams":1, "nb_programs":1, "format_name": "test", "format_long_name": "test format name", "duration":12.345}}`)
	mockBody := helpers.NewMockReadCloser()
	mockBody.DataToRead = mockRequestBody

	mockRequest := http.Request{
		Method:     "POST",
		RequestURI: "https://myserver.com/api/analysis/result?forJob=95C2E86F-C0C3-4D9F-B9E1-0AC878BE6B10&stepId=15B4342F-12EA-4986-9668-9943A153F280",
		Proto:      "https",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body:       mockBody,
	}

	jobMasterId := uuid.MustParse("95C2E86F-C0C3-4D9F-B9E1-0AC878BE6B10")
	jobStepId := uuid.MustParse("15B4342F-12EA-4986-9668-9943A153F280")
	startTime := time.Now()

	fakeJobContainer := models2.JobContainer{
		Id: jobMasterId,
		Steps: []models2.JobStep{
			models2.JobStepAnalysis{
				JobStepType: "analysis",
				JobStepId:   uuid.New(),
			},
			models2.JobStepThumbnail{
				JobStepType: "thumbnail",
				JobStepId:   jobStepId,
			},
		},
		CompletedSteps:    0,
		Status:            1,
		JobTemplateId:     uuid.New(),
		ErrorMessage:      "",
		IncomingMediaFile: "",
		StartTime:         &startTime,
	}

	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	testClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer func() {
		testClient.Close()
		s.Close()
	}()

	fakeJobContainer.Store(testClient)

	toTest := ReceiveData{redisClient: testClient}
	mockWriter := helpers.NewMockResponseWriter()
	toTest.ServeHTTP(mockWriter, &mockRequest)

	if len(mockWriter.State.LastWrittenBytes) == 0 {
		t.Error("Nothing was written to client")
		t.FailNow()
	}

	if mockWriter.State.WrittenStatusCode == nil {
		t.Error("No status code was written")
	} else {
		if *mockWriter.State.WrittenStatusCode != 404 {
			t.Errorf("Incorrect status code was written, expected 404 got %d", *mockWriter.State.WrittenStatusCode)
		}
	}

	jsonContent, jsonErr := mockWriter.LastWrittenJson()
	if jsonErr != nil {
		t.Error("Method did not output json: ", jsonErr)
	} else {
		if jsonContent["status"].(string) == "ok" {
			t.Error("status field was 'ok' and it should not be on error")
		}
		if !strings.Contains(jsonContent["detail"].(string), "not analysis") {
			t.Errorf("output detail message %s does not contain the string 'not analysis'", jsonContent["detail"].(string))
		}
		spew.Dump(jsonContent)
	}
}
