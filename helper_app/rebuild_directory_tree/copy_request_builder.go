package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
)

type CopyRequest struct {
	From string
	To string
}

type GenericResponseContainer struct {
	Status string `json:"status"`
	Count int `json:"count"`
	Entries []models.FileEntry `json:"entries"`
}

func lookupAssociatedFiles(baseUrl string, jobId uuid.UUID) (*[]models.FileEntry, error) {
	targetUrl := fmt.Sprintf("%s/api/file/byJob?jobId=%s", baseUrl, jobId.String())
	response, err := http.Get(targetUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	contentBytes, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}

	var parsedContent GenericResponseContainer
	unmarshalErr := json.Unmarshal(contentBytes, &parsedContent)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &(parsedContent.Entries), nil
}

func makeDestPath(sourceMediaFile string, outputBasePath string, targetPathStrip int) (string, error) {
	originalMediaPath := path.Dir(sourceMediaFile)
	if strings.HasPrefix(originalMediaPath, "/") {
		originalMediaPath = originalMediaPath[1:]
	}
	pathParts := strings.Split(originalMediaPath, "/")

	var partsToUse []string
	if len(pathParts)>= targetPathStrip {
		partsToUse = pathParts[targetPathStrip:]
	} else {
		return "", errors.New(fmt.Sprintf("the incoming path '%s' does not have enough segments to strip %d away", originalMediaPath, targetPathStrip))
	}

	return path.Join(outputBasePath, strings.Join(partsToUse, "/")), nil
}

/**
looks up the files associated with the incoming job and pushes CopyRequest objects onto the output channel
for each file
 */
func AsyncCopyRequestBuilder(inputCh chan *models.JobContainer, baseUrl string, outputBasePath string, targetPathStrip int, queueSize int) (chan *CopyRequest, chan error) {
	outputCh := make(chan *CopyRequest, queueSize)
	errCh := make(chan error, 1)

	go func() {
		for {
			rec := <- inputCh
			if rec==nil {
				log.Print("INFO AsyncCopyRequestBuilder reached end of stream")
				outputCh <- nil
				return
			}

			filesListPtr, getFilesErr := lookupAssociatedFiles(baseUrl, rec.Id)
			if getFilesErr != nil {
				log.Printf("WARNING AsyncCopyRequestBuilder could not get files for %s: %s", rec.Id, getFilesErr)
				continue
			}

			desiredDestPath, destPathErr := makeDestPath(rec.IncomingMediaFile, outputBasePath, targetPathStrip)
			if destPathErr != nil {
				log.Printf("ERROR AsyncCopyRequestBuilder could not determine destination path: %s", destPathErr)
				errCh <- destPathErr
				return
			}

			for _, file := range *filesListPtr {
				copyReq := &CopyRequest{
					From: file.ServerPath,
					To:   path.Join(desiredDestPath, path.Base(file.ServerPath)),
				}
				outputCh <- copyReq
			}
		}
	}()

	return outputCh, errCh
}
