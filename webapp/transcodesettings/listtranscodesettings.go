package transcodesettings

import (
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type ListTranscodeSettings struct {
	mgr *models.TranscodeSettingsManager
}

func (h ListTranscodeSettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	results := h.mgr.ListSettings()
	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"count":   len(*results),
		"entries": results,
	}, w, 200)
}
