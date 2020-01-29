package jobrunner

import (
	"testing"
)

func TestReadTemplate(t *testing.T) {
	result, err := LoadFromTemplate("../config/AnalysisJobTemplate.yaml")

	if err != nil {
		t.Error("Got an unexpected error, ", err, ", load should have succeeded")
	}

	if result.Name != "analysis-job-template" {
		t.Error("Got unexpected name ", result.Name)
	}

}
