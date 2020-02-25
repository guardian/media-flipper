package main

import (
	"fmt"
	"github.com/guardian/mediaflipper/common/models"
	"log"
	"os"
	"os/exec"
	"time"
)

func RunVideoThumbnail(fileName string, outPath string, atFrame int) *ThumbnailResult {
	outFileName := GetOutputFilenameThumb(outPath, fileName)

	cmd := exec.Command("ffmpeg", "-i", fileName, "-vframes", "1", "-an", "-y", "-ss", fmt.Sprint(atFrame), outFileName)

	return runThumbnailWrapper(cmd, outFileName)
}

func RunImageThumbnail(fileName string, outPath string, settings models.TranscodeTypeSettings) *ThumbnailResult {
	outFileName := GetOutputFilenameThumb(outPath, fileName)
	removeOnSuccess := false

	isRaw, checkErr := CheckIsRaw(fileName)
	if checkErr != nil {
		log.Printf("WARNING: raw image check for %s failed: %s", fileName, checkErr)
	}
	updatedFileName := fileName

	//if we have a RAW file, try to extract out the thumbnail jpeg embedded in the file
	if isRaw && checkErr == nil {
		startTime := time.Now()
		extractedFileName, extractErr := ExtractRawThumbnail(fileName)
		endTime := time.Now()
		if extractErr == nil { //if we got an embedded file, return that
			duration := endTime.UnixNano() - startTime.UnixNano()
			return &ThumbnailResult{
				OutPath:      &extractedFileName,
				ErrorMessage: nil,
				TimeTaken:    float64(duration) / 1e9,
			}
		} else { //if we could not get an embedded file, try to convert the whole thing
			log.Printf("WARNING: could not extract thumbnail from %s: %s", fileName, extractErr)
			var convErr error
			updatedFileName, convErr = RawToTiff(fileName)
			if convErr != nil {
				log.Printf("WARNING: could not convert file %s: %s", fileName, convErr)
				updatedFileName = fileName
			} else {
				removeOnSuccess = true
			}
		}
	}

	commandArgs := []string{updatedFileName}
	commandArgs = append(commandArgs, settings.MarshalToArray()...)
	commandArgs = append(commandArgs, outFileName)
	cmd := exec.Command("/usr/bin/convert", commandArgs...)

	result := runThumbnailWrapper(cmd, outFileName)
	if removeOnSuccess && result.ErrorMessage == nil {
		os.Remove(updatedFileName)
	}
	return result
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
