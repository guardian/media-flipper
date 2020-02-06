package jobtemplate

import (
	"github.com/guardian/mediaflipper/common/helpers"
	models2 "github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type ListTemplateHandler struct {
	templateMgr *models2.JobTemplateManager
}

func (h ListTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	helpers.WriteJsonContent(map[string]interface{}{
		"status":  "ok",
		"entries": h.templateMgr.ListTemplates(),
	}, w, 200)
}
