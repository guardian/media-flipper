package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func SendToWebapp(forUrl string, data interface{}, attempt int, maxTries int) error {
	byteData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		log.Print("ERROR: Could not marshal data for webapp send: ", marshalErr)
		return marshalErr
	}

	byteReader := bytes.NewReader(byteData)
	response, err := http.Post(forUrl, "application/json", byteReader)

	if err != nil {
		log.Print("ERROR: Could not send data to webapp: ", err)
		return err
	}

	responseContent, _ := ioutil.ReadAll(response.Body)
	switch response.StatusCode {
	case 500:
	case 503:
	case 504:
		log.Printf("WARNING: server said %s", string(responseContent))
		log.Printf("WARNING: Webapp is not accessible on attempt %d (got a %d response)", attempt, response.StatusCode)
		if attempt >= maxTries {
			return errors.New("Webapp was not accessible")
		}
		time.Sleep(1 * time.Second)
		return SendToWebapp(forUrl, data, attempt+1, maxTries)
	case 200:
	case 201:
		return nil
	default:
		log.Printf("ERROR: Webapp returned a fatal error (got a %d response)")
	}
	return errors.New("Got a fatal error, see logs")
}
