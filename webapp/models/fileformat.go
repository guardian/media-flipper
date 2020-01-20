package models

import (
	"github.com/google/uuid"
)

type FileFormatInfo struct {
	ForJob         uuid.UUID      `json:"forJob"`
	FormatAnalysis FormatAnalysis `json:"formatAnalysis"`
}
