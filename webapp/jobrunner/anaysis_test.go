package jobrunner

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

func TestReadTemplate(t *testing.T) {
	result, err := LoadFromTemplate("../config/AnalysisJobTemplate.yaml")

	if err != nil {
		t.Error("Got an unexpected error, ", err, ", load should have succeeded")
	}

	spew.Dump(result.Spec.Template.Spec.Volumes)

	if result.Name != "analysis-job-template" {
		t.Error("Got unexpected name ", result.Name)
	}

}
