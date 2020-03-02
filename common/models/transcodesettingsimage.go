package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type TranscodeImageSettings struct {
	Name        string    `json:"name" yaml:"name" mapstructure:"name"`
	Description string    `json:"description" yaml:"description" mapstructure:"description"`
	SettingsId  uuid.UUID `json:"settingsid" yaml:"settingsid" mapstructure:"settingsid"`
	ScaleX      int32     `json:"scale_x" mapstructure:"scale_x"`
	XMaxSize    bool      `json:"x_max_size" mapstructure:"x_max_size"` //if true, then X is a maximum size and the output can be smaller. if false then X is the exact size.
	ScaleY      int32     `json:"scale_y" mapstructure:"scale_y"`
	YMaxSize    bool      `json:"y_max_size" mapstructure:"y_max_size"`
}

func (s TranscodeImageSettings) MarshalToArray() []string {
	var xflag string
	if s.XMaxSize {
		xflag = ">"
	} else {
		xflag = ""
	}
	var yflag string
	if s.YMaxSize {
		yflag = ">"
	} else {
		yflag = ""
	}

	return []string{
		"-resize",
		fmt.Sprintf("%d%sx%d%s", s.ScaleX, xflag, s.ScaleY, yflag),
	}
}

func (s TranscodeImageSettings) MarshalToString() string {
	return strings.Join(s.MarshalToArray(), " ")
}

func (s TranscodeImageSettings) GetId() uuid.UUID {
	return s.SettingsId
}

func (s TranscodeImageSettings) Summarise() JobSettingsSummary {
	return JobSettingsSummary{
		SettingsId:  s.SettingsId,
		Name:        s.Name,
		Description: s.Description,
	}
}

func (s TranscodeImageSettings) InternalMarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s TranscodeImageSettings) IsValid() bool {
	return s.ScaleX > 0 && s.ScaleY > 0
}

func (s TranscodeImageSettings) GetLikelyExtension() string {
	return "jpg" //a reasonable assumption
}
