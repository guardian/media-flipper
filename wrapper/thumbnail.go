package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

func RunThumbnail(fileName string, atFrame int) *ThumbnailResult {
	outFileName := RemoveExtension(fileName) + "_thumb.jpg"
	startTime := time.Now()

	cmd := exec.Command("ffmpeg", "-i", "fileName", fileName, "-vframes", "1", "-an", "-ss", string(atFrame), outFileName)

	_, errContent, err := RunCommand(cmd)

	endTime := time.Now()

	duration := endTime.UnixNano() - startTime.UnixNano()

	if err != nil {
		log.Printf("Command failed")
		_, fileErr := os.Stat(outFileName)
		if os.IsNotExist(fileErr) {
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
