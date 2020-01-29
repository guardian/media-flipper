package jobtemplate

import (
	"github.com/guardian/mediaflipper/webapp/helpers"
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type ListTemplateHandler struct {
	templateMgr *models.JobTemplateManager
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
