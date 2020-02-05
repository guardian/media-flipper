package transcodesettings

import (
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type TranscodeSettingsEndpoints struct {
	getEndpoint  GetTranscodeSettings
	listEndpoint ListTranscodeSettings
}

func NewTranscodeSettingsEndpoints(mgr *models.TranscodeSettingsManager) TranscodeSettingsEndpoints {
	return TranscodeSettingsEndpoints{
		getEndpoint:  GetTranscodeSettings{mgr: mgr},
		listEndpoint: ListTranscodeSettings{mgr: mgr},
	}
}

func (t TranscodeSettingsEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"/get", t.getEndpoint)
	http.Handle(baseUrl, t.listEndpoint)
}
