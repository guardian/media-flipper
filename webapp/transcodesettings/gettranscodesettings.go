package transcodesettings

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"net/http"
	"net/url"
)

type GetTranscodeSettings struct {
	mgr *models.TranscodeSettingsManager
}

func (h GetTranscodeSettings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	requestUrl, _ := url.ParseRequestURI(r.RequestURI)
	uuidText := requestUrl.Query().Get("forId")
	sId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "error",
			Detail: "Invalid file ID",
		}, w, 400)
		return
	}

	s := h.mgr.GetSetting(sId)
	if s == nil {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "not_found",
			Detail: "nothing found with that ID",
		}, w, 404)
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status": "ok",
		"entry":  s,
	}, w, 200)
}
