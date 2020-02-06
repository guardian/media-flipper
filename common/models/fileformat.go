package models

import (
	"github.com/google/uuid"
)

type FileFormatInfo struct {
	Id             uuid.UUID      `json:"id"`
	FormatAnalysis FormatAnalysis `json:"formatAnalysis"`
}
