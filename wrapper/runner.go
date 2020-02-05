package main

import (
	"io"
	"io/ioutil"
	"log"
	"os/exec"
)

/**
helper function to run the given command and capture output
*/
func RunCommand(cmd *exec.Cmd) ([]byte, []byte, error) {
	log.Print("DEBUG: exec command is ", cmd)
	outPipe, _ := cmd.StdoutPipe()
	errPipe, _ := cmd.StderrPipe()

	startErr := cmd.Start()
	if startErr != nil {
		log.Print("Could not start command: ", startErr)
		return nil, nil, startErr
	}

	outContent, _ := ioutil.ReadAll(outPipe)
	errContent, _ := ioutil.ReadAll(errPipe)

	completeErr := cmd.Wait()
	if completeErr != nil {
		exitErr, isExitError := completeErr.(*exec.ExitError)
		if isExitError {
			log.Print("Failure code: ", exitErr)
			log.Printf("Subprocess exited with an error: \n%s\n%s", exitErr.Stderr, errContent)
			return outContent, errContent, completeErr
		} else {
			log.Print("Could not run subprocess: ", completeErr)
			return outContent, errContent, completeErr
		}
	}

	return outContent, errContent, nil
}

func pipeToChan(reader io.Reader, channel chan string) {
	for {
		buf := make([]byte, 1024)
		bytesRead, err := reader.Read(buf)
		if err != nil {
			close(channel)
			if err == io.EOF {
				log.Print("End of stream reached")
				return
			}
			log.Print("ERROR: could not read: ", err)
			return
		}
		finalBytes := make([]byte, bytesRead)
		copy(finalBytes, buf)
		channel <- string(finalBytes)
	}
}

func RunCommandStreaming(cmd *exec.Cmd) (chan string, chan string, error) {
	outPipe, _ := cmd.StdoutPipe()
	errPipe, _ := cmd.StderrPipe()

	outChan := make(chan string, 100)
	errChan := make(chan string, 100)

	go pipeToChan(outPipe, outChan)
	go pipeToChan(errPipe, errChan)

	startErr := cmd.Start()
	if startErr != nil {
		return nil, nil, startErr
	} else {
		return outChan, errChan, nil
	}
}
