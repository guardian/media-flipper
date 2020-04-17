package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

//func PostWithTimeout(url string, contentType string, body io.Reader, timeout time.Duration) (*http.Response, error) {
//	responseChan := make(chan *http.Response)
//	errChan := make(chan error)
//	timer := time.NewTimer(timeout)
//
//	defer func() {
//		//ensure that all channels are cleaned up
//		close(responseChan)
//		close(errChan)
//		timer.Stop()
//	}()
//
//	go func() {
//		response, err := http.Post(url, contentType, body)
//		if err != nil {
//			errChan <- err
//		} else {
//			responseChan <- response
//		}
//	}()
//
//	select { //we only expect a message on one of the channels
//	case response := <-responseChan:
//		log.Printf("INFO PostWithTimeout http send was successful")
//		return response, nil
//	case err := <-errChan:
//		log.Printf("INFO PostWithTimeout http send failed, see caller for error details")
//		return nil, err
//	case <-timer.C:
//		err := TimeoutError{
//			what:   fmt.Sprintf("http post to %s", url),
//			expiry: timeout,
//		}
//		log.Printf("WARNING: PostWithTimeout failed: %s", err.Error())
//		return nil, err
//	}
//}

func PostWithTimeout(url string, contentType string, byteData []byte, timeout time.Duration) (*http.Response, error) {
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	cancellationContext, callToCancel := context.WithCancel(context.Background())

	byteReader := bytes.NewReader(byteData)
	rq, buildErr := http.NewRequest(url, contentType, byteReader)
	if buildErr != nil {
		log.Printf("ERROR: PostWithTimeout could not build request: %s", buildErr)
		return nil, buildErr
	}

	timer := time.NewTimer(timeout)
	requestDone := make(chan interface{})
	var didCancel = false
	go func() {
		//this is a watchdog that will cancel the request if if takes too long
		select {
		case <-requestDone:
			timer.Stop()
		case <-timer.C:
			log.Printf("WARNING SendToWebApp request timed out after 30s, re-sending")
			didCancel = true
			callToCancel()
			close(requestDone)
		}
	}()

	response, err := client.Do(rq.WithContext(cancellationContext))

	if didCancel {
		log.Printf("WARNING SendToApp re-sending request")
		time.Sleep(3 * time.Second)
		return PostWithTimeout(url, contentType, byteData, timeout)
	} else {
		requestDone <- true
		log.Printf("DEBUG SendToApp success")
		return response, err
	}
}

//taken (with thanks!) from https://stackoverflow.com/questions/29197685/how-to-close-abort-a-golang-http-client-post-prematurely
//func httpDo(ctx context.Context, req *http.Request, f func(*http.Response, error) error) error {
//	// Run the HTTP request in a goroutine and pass the response to f.
//	tr := &http.Transport{}
//	client := &http.Client{Transport: tr}
//	c := make(chan error, 1)
//	go func() { c <- f(client.Do(req)) }()
//	select {
//	case <-ctx.Done():
//		tr.CancelRequest(req)
//		<-c // Wait for f to return.
//		return ctx.Err()
//	case err := <-c:
//		return err
//	}
//}

func SendToWebapp(forUrl string, data interface{}, attempt int, maxTries int) error {
	byteData, marshalErr := json.Marshal(data)
	if marshalErr != nil {
		log.Print("ERROR: Could not marshal data for webapp send: ", marshalErr)
		return marshalErr
	}

	response, err := PostWithTimeout(forUrl, "application/json", byteData, 30*time.Second)

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
			return errors.New("webapp was not accessible")
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
	return errors.New("got a fatal error, see logs")
}
