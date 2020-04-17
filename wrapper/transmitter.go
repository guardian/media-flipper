package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type TimeoutError struct {
	what   string
	expiry time.Duration
}

func (t TimeoutError) Error() string {
	return fmt.Sprintf("%s timed out after %s", t.what, t.expiry.String())
}

func PostWithTimeout(url string, contentType string, body io.Reader, timeout time.Duration) (*http.Response, error) {
	responseChan := make(chan *http.Response)
	errChan := make(chan error)
	timer := time.NewTimer(timeout)

	defer func() {
		//ensure that all channels are cleaned up
		close(responseChan)
		close(errChan)
		timer.Stop()
	}()

	go func() {
		response, err := http.Post(url, contentType, body)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	}()

	select { //we only expect a message on one of the channels
	case response := <-responseChan:
		log.Printf("INFO PostWithTimeout http send was successful")
		return response, nil
	case err := <-errChan:
		log.Printf("INFO PostWithTimeout http send failed, see caller for error details")
		return nil, err
	case <-timer.C:
		err := TimeoutError{
			what:   fmt.Sprintf("http post to %s", url),
			expiry: timeout,
		}
		log.Printf("WARNING: PostWithTimeout failed: %s", err.Error())
		return nil, err
	}
}

func SendToWebapp(forUrl string, data interface{}, attempt int, maxTries int) error {
	byteData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		log.Print("ERROR: Could not marshal data for webapp send: ", marshalErr)
		return marshalErr
	}

	byteReader := bytes.NewReader(byteData)
	response, err := PostWithTimeout(forUrl, "application/json", byteReader, 3*time.Second)

	if err != nil {
		log.Print("ERROR: Could not send data to webapp: ", err)
		//backoff and retry if the http request times out
		if _, isTimeout := err.(TimeoutError); isTimeout {
			if attempt >= maxTries {
				return errors.New("Webapp was not accessible")
			}
			log.Printf("WARNING: Send to webapp timed out. Retrying after 3s delay...")
			time.Sleep(3 * time.Second)
			return SendToWebapp(forUrl, data, attempt+1, maxTries)
		}
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
		fallthrough
	case 201:
		return nil
	default:
		log.Printf("ERROR: Webapp returned a fatal error (got a %d response)", response.StatusCode)
		log.Printf("ERROR: Server said %s", responseContent)
	}
	return errors.New("Got a fatal error, see logs")
}
