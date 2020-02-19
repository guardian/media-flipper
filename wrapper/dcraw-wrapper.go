package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

/**
run dcraw -i (info) and check return code to see if the given file is processable as raw
*/
func CheckIsRaw(filename string) (bool, error) {
	cmd := exec.Command("dcraw", "-i", filename)
	startErr := cmd.Start()
	if startErr != nil {
		return false, startErr
	}
	waitErr := cmd.Wait()
	if waitErr == nil {
		return true, nil //process ran successfully => dcraw can read it
	}
	exErr, isExitErr := waitErr.(*exec.ExitError)

	if isExitErr {
		log.Print(string(exErr.Stderr)) //nonzero exit code => dcraw can't read it
		return false, nil
	} else {
		return false, waitErr
	}
}

func ExtractRawThumbnail(filename string) (string, error) {
	expectedOutputFilename := fmt.Sprintf("%s.thumb.jpg", RemoveExtension(filename))

	cmd := exec.Command("dcraw", "-e", filename)
	stdOutBytes, stdErrBytes, err := RunCommand(cmd)
	log.Print(string(stdOutBytes))
	log.Print(string(stdErrBytes))

	if err != nil {
		return "", err
	}

	_, statErr := os.Stat(expectedOutputFilename)
	if statErr != nil {
		log.Printf("Could not find expected output file at %s, assuming failure", expectedOutputFilename)
		return "", errors.New("no output file was created")
	} else {
		return expectedOutputFilename, nil
	}
}

func RawToTiff(filename string) (string, error) {
	expectedOutputFilename := fmt.Sprintf("%s.tiff", RemoveExtension(filename))

	cmd := exec.Command("dcraw", "-e", filename)
	stdOutBytes, stdErrBytes, err := RunCommand(cmd)
	log.Print(string(stdOutBytes))
	log.Print(string(stdErrBytes))

	if err != nil {
		return "", err
	}

	_, statErr := os.Stat(expectedOutputFilename)
	if statErr != nil {
		log.Printf("Could not find expected output file at %s, assuming failure", expectedOutputFilename)
		return "", errors.New("no output file was created")
	} else {
		return expectedOutputFilename, nil
	}
}
