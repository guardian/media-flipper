package jobrunner

import (
	"errors"
	"github.com/google/uuid"
	"testing"
)

func TestCreateGenericJobInternal(t *testing.T) {
	mockClient := &JobClientMock{}

	stepId := uuid.New()
	envVars := map[string]string{
		"VAR_ONE": "value 1",
		"VAR_TWO": "value 2",
	}
	result := createGenericJobInternal(stepId, "test-fake-job", envVars, true, "../config/AnalysisJobTemplate.yaml", mockClient)

	if result != nil {
		t.Errorf("createGenericJobInternal raised unexpected error: %s", result)
	} else {
		if len(mockClient.JobsCreated) != 1 {
			t.Errorf("job create was called %d times, expected 1", len(mockClient.JobsCreated))
		}
		if len(mockClient.JobsCreated) > 0 {
			j := *mockClient.JobsCreated[0]
			if len(j.Spec.Template.Spec.Containers) == 0 {
				t.Errorf("Created job had no containers, this is not right")
			} else {
				c := j.Spec.Template.Spec.Containers[0]
				if len(c.Env) != len(envVars) {
					t.Errorf("Expected %d environment variables but got %d", len(envVars), len(c.Env))
				}
				for _, v := range c.Env {
					setValue, hasKey := envVars[v.Name]
					if !hasKey {
						t.Errorf("Environment variables missing set value %s", v.Name)
					} else {
						if setValue != v.Value {
							t.Errorf("Environment variable %s set incorrectly. Expected %s, got %s", v.Name, setValue, v.Value)
						}
					}
				}
			}

			labels := j.GetLabels()
			flipperId, gotFlipperId := labels["mediaflipper.jobStepId"]
			if gotFlipperId {
				if flipperId != stepId.String() {
					t.Errorf("created pod had wrong job id label, expected %s got %s", stepId.String(), flipperId)
				}
			} else {
				t.Errorf("created pod was missing mediaflipper.jobStepId label")
			}
		}
	}

	testError := errors.New("kaboom!")
	mockFailingClient := &JobClientMock{
		ErrorResponse: testError,
	}

	failedResult := createGenericJobInternal(stepId, "test-failing-job", envVars, true, "../config/AnalysisJobTemplate.yaml", mockFailingClient)
	if failedResult == nil {
		t.Errorf("expected createGenericJobInternal to fail if create operation fails, but it returned no error")
	}

	noTemplateResult := createGenericJobInternal(stepId, "test-failing-job", envVars, true, "fsfsjkhdfjsdfs", mockClient)
	if noTemplateResult == nil {
		t.Errorf("expected createGenericJobInternal to fail if the template could not be found, but it returned no error")
	}
}
