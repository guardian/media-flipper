package helpers

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func WriteJsonContent(content interface{}, w http.ResponseWriter, statusCode int) {
	contentBytes, marshalErr := json.Marshal(content)
	if marshalErr != nil {
		log.Printf("Could not marshal content for json write: %s", marshalErr)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", strconv.FormatInt(int64(len(contentBytes)), 10))
	w.WriteHeader(statusCode)
	_, writeErr := w.Write(contentBytes)
	if writeErr != nil {
		log.Printf("Could not write content to HTTP socket: %s", writeErr)
	}
}

func ReadJsonBody(from io.Reader, to interface{}) error {
	byteContent, readErr := ioutil.ReadAll(from)
	if readErr != nil {
		return readErr
	}

	marshalErr := json.Unmarshal(byteContent, to)
	return marshalErr
}

func AssertHttpMethod(request *http.Request, w http.ResponseWriter, method string) bool {
	if request.Method != method {
		log.Printf("Got a %s request, expecting %s", request.Method, method)
		WriteJsonContent(GenericErrorResponse{"error", "wrong method type"}, w, 405)
		return false
	} else {
		return true
	}
}

/**
Breaks down the incoming request URI into a map of string->string
*/
func GetQueryParams(incomingRequestUri string) (*url.Values, error) {
	requestUri, uriParseErr := url.ParseRequestURI(incomingRequestUri)

	if uriParseErr != nil {
		log.Printf("Could not understand incoming request URI '%s': %s", incomingRequestUri, uriParseErr)
		return nil, errors.New("Invalid URI")
	}

	rtn := requestUri.Query()
	return &rtn, nil
}

/**
gets just the "JobID" parameter from the provided query string and returns it as a pointer to UUID
if it does not exist or is not a valid UUID, a GenericErrorResponse object is returned that is suitable
to be written directly to the outgoing response.
This is a convenience function that calls GetQueryParams and GetJobIdFromValues
*/
func GetJobIdFromQuerystring(incomingRequestUri string) (*uuid.UUID, *GenericErrorResponse) {
	queryParams, err := GetQueryParams(incomingRequestUri)
	if err != nil {
		return nil, &GenericErrorResponse{
			Status: "error",
			Detail: err.Error(),
		}
	}
	return GetJobIdFromValues(queryParams)
}

func GetJobIdFromValues(queryParams *url.Values) (*uuid.UUID, *GenericErrorResponse) {
	jobIdString := queryParams.Get("jobId")

	jobId, uuidParseErr := uuid.Parse(jobIdString)
	if uuidParseErr != nil {
		log.Printf("Could not parse job ID string '%s' into a UUID: %s", jobIdString, uuidParseErr)
		return nil, &GenericErrorResponse{
			Status: "error",
			Detail: "malformed UUID",
		}
	}
	return &jobId, nil
}
