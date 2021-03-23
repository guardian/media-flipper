package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/guardian/mediaflipper/common/models"
	"io/ioutil"
	"log"
	"net/http"
)

type ListJobResponse struct {
	Status     string                 `json:"status"`
	NextCursor uint64                 `json:"nextCursor"`
	Entries    *[]models.JobContainer `json:"entries"`
}

func getNextPage(httpClient *http.Client, baseUrl string, startAt int, pageSize int) (*[]models.JobContainer, error) {
	targetUrl := fmt.Sprintf("%s/api/job?startindex=%d&limit=%d", baseUrl, startAt, startAt+pageSize)
	response, err := httpClient.Get(targetUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	bodyContent, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}

	var parsedContent ListJobResponse
	marshalErr := json.Unmarshal(bodyContent, &parsedContent)
	if marshalErr != nil {
		return nil, marshalErr
	}

	return parsedContent.Entries, nil
}

func AsyncScanJobs(baseUrl string, pageSize int) (chan *models.JobContainer, chan error) {
	outputCh := make(chan *models.JobContainer, pageSize)
	errCh := make(chan error, 1)

	go func() {
		loaded := 0
		httpClient := &http.Client{}
		for {
			results, loadErr := getNextPage(httpClient, baseUrl, loaded, pageSize-1)
			if loadErr != nil {
				log.Print("ERROR AsyncScanJobs could not read page: ", loadErr)
				errCh <- loadErr
				return
			}
			if results == nil {
				log.Print("ERROR AsyncScanJobs received empty data")
				errCh <- errors.New("no data returned")
				return
			}
			if len(*results) == 0 {
				log.Print("INFO AsyncScanJobs completed scan")
				outputCh <- nil
				return
			}

			loaded += len(*results)
			log.Printf("DEBUG AsyncScanJobs got page of %d results for %d total", len(*results), loaded)
			for _, entry := range *results {
				copiedEntry := entry
				outputCh <- &copiedEntry
			}
		}
	}()

	return outputCh, errCh
}
