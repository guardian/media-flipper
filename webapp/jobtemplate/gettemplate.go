package jobtemplate

import (
	"github.com/guardian/mediaflipper/common/helpers"
	"github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type GetTemplate struct {
	templateMgr *models.JobTemplateManager
}

func (h GetTemplate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !helpers.AssertHttpMethod(r, w, "GET") {
		return
	}

	_, templateId, err := helpers.GetForId(r.RequestURI)
	if err != nil {
		helpers.WriteJsonContent(err, w, 400)
		return
	}

	tpl, foundTpl := h.templateMgr.GetJob(*templateId)
	if foundTpl {
		helpers.WriteJsonContent(map[string]interface{}{
			"status": "ok",
			"entry":  tpl,
		}, w, 200)
	} else {
		helpers.WriteJsonContent(helpers.GenericErrorResponse{"not_found", "no template with that id"}, w, 404)
	}
}
