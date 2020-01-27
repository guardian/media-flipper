package jobtemplate

import (
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type TemplateEndpoints struct {
	listHandler ListTemplateHandler
}

func NewTemplateEndpoints(jobTemplateMgr *models.JobTemplateManager) TemplateEndpoints {
	return TemplateEndpoints{
		listHandler: ListTemplateHandler{templateMgr: jobTemplateMgr},
	}
}

func (e TemplateEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"", e.listHandler)
}
