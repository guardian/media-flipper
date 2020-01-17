package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os/exec"
)

func RunAnalysis(fileName string) (*AnalysisResult, error) {
	cmd := exec.Command("ffprobe", "-of", "json", "-show_format", "-show_streams", "-show_programs", fileName)

	outPipe, _ := cmd.StdoutPipe()

	startErr := cmd.Start()
	if startErr != nil {
		log.Print("Could not start analysis command: ", startErr)
		return nil, startErr
	}

	outContent, _ := ioutil.ReadAll(outPipe)

	completeErr := cmd.Wait()
	if completeErr != nil {
		exitErr, isExitError := completeErr.(*exec.ExitError)
		if isExitError {
			log.Print("Failure code: ", exitErr)
			log.Printf("Subprocess exited with an error: \n%s\n%s", exitErr.Stderr)
			return nil, completeErr
		} else {
			log.Print("Could not run subprocess: ", completeErr)
			return nil, completeErr
		}
	}

	var rawOutput map[string]interface{}

	unmarshalErr := json.Unmarshal(outContent, &rawOutput)
	if unmarshalErr != nil {
		log.Print("Offending content was ", string(outContent))
		log.Print("Could not unmarshal content from subprocess: ", unmarshalErr)
		return nil, unmarshalErr
	}

	//log.Print("debug: raw output is ", rawOutput["format"].(map[string]interface{}))
	//
	//var formatInfo FormatAnalysis
	//err := mapstructure.Decode(rawOutput["format"].(map[string]interface{}), &formatInfo)
	//
	//if err != nil {
	//	log.Fatal(err)
	//}
	return &AnalysisResult{Success: true, Format: FormatAnalysisFromMap(rawOutput["format"].(map[string]interface{}))}, nil

}
