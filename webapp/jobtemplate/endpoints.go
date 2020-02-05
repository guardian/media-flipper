package jobtemplate

import (
	"github.com/guardian/mediaflipper/webapp/models"
	"net/http"
)

type TemplateEndpoints struct {
	listHandler ListTemplateHandler
	getHandler  GetTemplate
}

func NewTemplateEndpoints(jobTemplateMgr *models.JobTemplateManager) TemplateEndpoints {
	return TemplateEndpoints{
		listHandler: ListTemplateHandler{templateMgr: jobTemplateMgr},
		getHandler:  GetTemplate{templateMgr: jobTemplateMgr},
	}
}

func (e TemplateEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"", e.listHandler)
	http.Handle(baseUrl+"/get", e.getHandler)
}
