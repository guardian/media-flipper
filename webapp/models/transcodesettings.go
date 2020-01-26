package models

import (
	"github.com/google/uuid"
)

type VideoSettings struct {
	Codec   string `json:"codec"`
	Bitrate string `json:"bitrate"`
	Scale   string `json:"scale"` //can be empty string
}

type AudioSettings struct {
	Codec   string `json:"codec"`
	Bitrate string `json:"bitrate"`
}

type WrapperSettings struct {
	Format string `json:"format"`
}

type JobSettings struct {
	SettingsId  uuid.UUID       `json:"settingsId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Video       VideoSettings   `json:"video"`
	Audio       AudioSettings   `json:"audio"`
	Wrapper     WrapperSettings `json:"format"`
}
