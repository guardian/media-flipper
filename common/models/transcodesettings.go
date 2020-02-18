package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"strconv"
	"strings"
)

type TranscodeTypeSettings interface {
	MarshalToString() string
	MarshalToArray() []string
	GetId() uuid.UUID
	Summarise() JobSettingsSummary
	InternalMarshalJSON() ([]byte, error)
	isValid() bool
}

type ScaleSettings struct {
	ScaleX         int32 `json:"scalex" yaml:"scalex"`                 //set to -1 for "preserve aspect"
	ScaleY         int32 `json:"scaley" yaml:"scaley"`                 //set to -1 for "preserve aspect"
	AllowUpscaling bool  `json:"allowupscaling" yaml:"allowupscaling"` //default false. if set allows the result to be made bigger
}

type VideoSettings struct {
	Codec   string         `json:"codec" yaml:"codec"`
	Bitrate int64          `json:"bitrate" yaml:"bitrate"` //in BYTES per sec. Or specify CRF...
	CRF     int8           `json:"crf" yaml:"crf"`         //A lower value generally leads to higher quality, and a subjectively sane range is 17–28. Consider 17 or 18 to be visually lossless or nearly so
	Preset  string         `json:"preset" yaml:"preset"`   //see https://trac.ffmpeg.org/wiki/Encode/H.264
	Scale   *ScaleSettings `json:"scale" yaml:"scale"`
}

type AudioSettings struct {
	Codec      string `json:"codec" yaml:"codec"`
	Bitrate    int64  `json:"bitrate" yaml:"bitrate"` //in BYTES per sec
	Channels   int8   `json:"channels" yaml:"channels"`
	Samplerate int32  `json:"samplerate" yaml:"samplerate"`
}

type WrapperSettings struct {
	Format string `json:"format" yaml:"format"`
}

type JobSettings struct {
	SettingsId  uuid.UUID       `json:"settingsid" yaml:"settingsid" mapstructure:"settingsid"`
	Name        string          `json:"name" yaml:"name" mapstructure:"name"`
	Description string          `json:"description" yaml:"description" mapstructure:"description"`
	Video       VideoSettings   `json:"video" yaml:"video" mapstructure:"video"`
	Audio       AudioSettings   `json:"audio" yaml:"audio" mapstructure:"audio"`
	Wrapper     WrapperSettings `json:"wrapper" yaml:"wrapper" mapstructure:"wrapper"`
}

type JobSettingsSummary struct {
	SettingsId  uuid.UUID `json:"settingsid" yaml:"settingsid"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
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
		//"-pix_fmt yuv420p",
	}
	if v.CRF > 0 {
		if v.CRF < 17 || v.CRF > 28 {
			log.Printf("WARNING: Provided CRF value %d is outside the recommended range of 17-28. See https://trac.ffmpeg.org/wiki/Encode/H.264", v.CRF)
		}
		out = append(out, "-crf", strconv.FormatInt(int64(v.CRF), 10))
	} else {
		out = append(out, "-b:v", strconv.FormatInt(v.Bitrate, 10))
	}
	if v.Preset != "" {
		out = append(out, "-preset", v.Preset)
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
		"-b:a",
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

func (s JobSettings) GetId() uuid.UUID {
	return s.SettingsId
}

func (s JobSettings) InternalMarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s JobSettings) isValid() bool {
	return s.Wrapper.Format != "" //so long as we have a wrapper format the settings can be used
}
