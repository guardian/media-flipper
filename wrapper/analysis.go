package main

import (
	"encoding/json"
	"log"
	"os/exec"
)

func RunAnalysis(fileName string) (*AnalysisResult, error) {
	cmd := exec.Command("ffprobe", "-of", "json", "-show_format", "-show_streams", "-show_programs", fileName)

	outContent, _, err := RunCommand(cmd)
	if err != nil {
		return nil, err
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
