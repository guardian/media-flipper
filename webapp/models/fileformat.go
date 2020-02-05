package models

import (
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
)

type FileFormatInfo struct {
	Id             uuid.UUID             `json:"id"`
	FormatAnalysis models.FormatAnalysis `json:"formatAnalysis"`
}
