package models

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

type TranscodeTypeSettings interface {
	MarshalToString() string
	MarshalToArray() []string
}

type ScaleSettings struct {
	ScaleX         int32 `json,yaml:"scalex"`         //set to -1 for "preserve aspect"
	ScaleY         int32 `json,yaml:"scaley"`         //set to -1 for "preserve aspect"
	AllowUpscaling bool  `json,yaml:"allowupscaling"` //default false. if set allows the result to be made bigger
}

type VideoSettings struct {
	Codec   string         `json,yaml:"codec"`
	Bitrate int64          `json,yaml:"bitrate"` //in BYTES per sec
	Scale   *ScaleSettings `json,yaml:"scale"`
}

type AudioSettings struct {
	Codec      string `json,yaml:"codec"`
	Bitrate    int64  `json,yaml:"bitrate"` //in BYTES per sec
	Channels   int8   `json,yaml:"channels"`
	Samplerate int32  `json,yaml:"samplerate"`
}

type WrapperSettings struct {
	Format string `json,yaml:"format"`
}

type JobSettings struct {
	SettingsId  uuid.UUID       `json,yaml:"settingsid"`
	Name        string          `json,yaml:"name"`
	Description string          `json,yaml:"description"`
	Video       VideoSettings   `json,yaml:"video"`
	Audio       AudioSettings   `json,yaml:"audio"`
	Wrapper     WrapperSettings `json,yaml:"format"`
}

type JobSettingsSummary struct {
	SettingsId  uuid.UUID `json,yaml:"settingsid"`
	Name        string    `json,yaml:"name"`
	Description string    `json,yaml:"description"`
}

func (s ScaleSettings) MarshalToString() string {
	if s.AllowUpscaling {
		return fmt.Sprintf("scale=%d:%d", s.ScaleX, s.ScaleY)
	} else {
		return fmt.Sprintf("scale='min(%d,iw):min(%d,ih)'", s.ScaleX, s.ScaleY)
	}
}

func (s ScaleSettings) MarshalToArray() []string {
	return []string{s.MarshalToString()}
}

func (v VideoSettings) MarshalToArray() []string {
	out := []string{
		"-vcodec",
		v.Codec,
		"-b:v",
		strconv.FormatInt(v.Bitrate, 10),
	}
	if v.Scale != nil {
		out = append(out, "-vf", v.Scale.MarshalToString())
	}
	return out
}

func (v VideoSettings) MarshalToString() string {
	return strings.Join(v.MarshalToArray(), " ")
}

func (a AudioSettings) MarshalToArray() []string {
	return []string{
		"-acodec",
		a.Codec,
		"-b:v",
		strconv.FormatInt(a.Bitrate, 10),
		"-ac",
		strconv.FormatInt(int64(a.Channels), 10),
		"-ar",
		strconv.FormatInt(int64(a.Samplerate), 10),
	}
}

func (a AudioSettings) MarshalToString() string {
	return strings.Join(a.MarshalToArray(), " ")
}

func (w WrapperSettings) MarshalToArray() []string {
	return []string{
		"-f",
		w.Format,
	}
}

func (w WrapperSettings) MarshalToString() string {
	return fmt.Sprintf("-f %s", w.Format)
}

func (s JobSettings) MarshalToString() string {
	settingsArray := []string{
		s.Video.MarshalToString(),
		s.Audio.MarshalToString(),
		s.Wrapper.MarshalToString(),
	}
	return strings.Join(settingsArray, " ")
}

func (s JobSettings) MarshalToArray() []string {
	result := append(s.Video.MarshalToArray(), s.Audio.MarshalToArray()...)
	result = append(result, s.Wrapper.MarshalToArray()...)
	return result
}

func (s JobSettings) Summarise() JobSettingsSummary {
	return JobSettingsSummary{
		SettingsId:  s.SettingsId,
		Name:        s.Name,
		Description: s.Description,
	}
}
