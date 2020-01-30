package models

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

type ScaleSettings struct {
	ScaleX         int32 //set to -1 for "preserve aspect"
	ScaleY         int32 //set to -1 for "preserve aspect"
	AllowUpscaling bool  //default false. if set allows the result to be made bigger
}

type VideoSettings struct {
	Codec   string         `json:"codec"`
	Bitrate int64          `json:"bitrate"` //in BYTES per sec
	Scale   *ScaleSettings `json:"scale"`   //can be empty string
}

type AudioSettings struct {
	Codec      string `json:"codec"`
	Bitrate    int64  `json:"bitrate"` //in BYTES per sec
	Channels   int8   `json:"channels"`
	Samplerate int32  `json:"samplerate"`
}

type WrapperSettings struct {
	Format string `json:"format"`
}

type TranscodeTypeSettings interface {
	MarshalToString() string
	MarshalToArray() []string
}

type JobSettings struct {
	SettingsId  uuid.UUID       `json:"settingsId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Video       VideoSettings   `json:"video"`
	Audio       AudioSettings   `json:"audio"`
	Wrapper     WrapperSettings `json:"format"`
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
