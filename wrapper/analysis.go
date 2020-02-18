package main

import (
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
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

	log.Printf("DEBUG: analysis result was: %s", spew.Sdump(rawOutput))
	log.Printf("DEBUG: format result was: %s", spew.Sdump(rawOutput["format"]))
	return &AnalysisResult{Success: true, Format: FormatAnalysisFromMap(rawOutput["format"].(map[string]interface{}))}, nil

}
