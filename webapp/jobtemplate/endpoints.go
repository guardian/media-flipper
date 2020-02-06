package jobtemplate

import (
	models2 "github.com/guardian/mediaflipper/common/models"
	"net/http"
)

type TemplateEndpoints struct {
	listHandler ListTemplateHandler
}

func NewTemplateEndpoints(jobTemplateMgr *models2.JobTemplateManager) TemplateEndpoints {
	return TemplateEndpoints{
		listHandler: ListTemplateHandler{templateMgr: jobTemplateMgr},
	}
}

func (e TemplateEndpoints) WireUp(baseUrl string) {
	http.Handle(baseUrl+"", e.listHandler)
}
