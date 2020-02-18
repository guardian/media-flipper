package main

import (
	"fmt"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"os"
	"os/exec"
	"time"
)

func RunVideoThumbnail(fileName string, atFrame int) *ThumbnailResult {
	outFileName := RemoveExtension(fileName) + "_thumb.jpg"

	cmd := exec.Command("ffmpeg", "-i", fileName, "-vframes", "1", "-an", "-y", "-ss", fmt.Sprint(atFrame), outFileName)

	return runThumbnailWrapper(cmd, outFileName)
}

func RunImageThumbnail(fileName string, settings models.TranscodeTypeSettings) *ThumbnailResult {
	outFileName := RemoveExtension(fileName) + "_thumb.jpg"

	commandArgs := []string{fileName}
	commandArgs = append(commandArgs, settings.MarshalToArray()...)
	commandArgs = append(commandArgs, outFileName)
	cmd := exec.Command("/usr/bin/convert", commandArgs...)

	return runThumbnailWrapper(cmd, outFileName)
}

func runThumbnailWrapper(cmd *exec.Cmd, outFileName string) *ThumbnailResult {
	startTime := time.Now()
	_, errContent, err := RunCommand(cmd)

	endTime := time.Now()

	duration := endTime.UnixNano() - startTime.UnixNano()

	if err != nil {
		log.Printf("Command failed")
		_, fileErr := os.Stat(outFileName)
		if !os.IsNotExist(fileErr) {
			log.Printf("Removing intermediate file %s", outFileName)
			os.Remove(outFileName)
		}
		errContentString := string(errContent)
		return &ThumbnailResult{
			OutPath:      nil,
			ErrorMessage: &errContentString,
			TimeTaken:    float64(duration) / 1e9,
		}
	}

	return &ThumbnailResult{
		OutPath:      &outFileName,
		ErrorMessage: nil,
		TimeTaken:    float64(duration) / 1e9,
	}
}
