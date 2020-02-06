package helpers

import (
	"github.com/google/uuid"
	"log"
	"net/url"
)

/**
helper function that parses the given URI and tries to extract uuid parameters with names "forJob" and "stepId"
returns:
 - pointer to the parsed URL or nil if it didn't parse
 - pointer to a uuid object corresponding to the "forJob" parameter or nil if it didn't parse
 - pointer to a uuid object corresponding to the "stepId" parameter or nil if it didn't parse
 - pointer to a GenericErrorResponse object if an error occurred
*/
func GetReceiverJobIds(uriString string) (*url.URL, *uuid.UUID, *uuid.UUID, *GenericErrorResponse) {
	requestUrl, urlErr := url.ParseRequestURI(uriString)
	if urlErr != nil {
		log.Print("requestURI could not parse, this should not happen: ", urlErr)
		return nil, nil, nil, &GenericErrorResponse{
			Status: "server_error",
			Detail: "requestUri could not parse, this should not happen",
		}
	}

	uuidText := requestUrl.Query().Get("forJob")
	jobContainerId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		log.Printf("Could not parse forJob parameter %s into a UUID: %s", uuidText, uuidErr)
		return requestUrl, nil, nil, &GenericErrorResponse{"error", "Invalid forJob parameter"}
	}

	jobStepText := requestUrl.Query().Get("stepId")
	jobStepId, uuidErr := uuid.Parse(jobStepText)

	if uuidErr != nil {
		log.Printf("Could not parse stepId parameter %s into a UUID: %s", jobStepText, uuidErr)
		return requestUrl, &jobContainerId, nil, &GenericErrorResponse{"error", "Invalid stepId parameter"}
	}

	return requestUrl, &jobContainerId, &jobStepId, nil
}

/**
helper function that parses the given URI and tries to extract uuid parameters with names "forId"
returns:
 - pointer to the parsed URL or nil if it didn't parse
 - pointer to a uuid object corresponding to the "forId" parameter or nil if it didn't parse
 - pointer to a GenericErrorResponse object if an error occurred
*/
func GetForId(uriString string) (*url.URL, *uuid.UUID, *GenericErrorResponse) {
	requestUrl, urlErr := url.ParseRequestURI(uriString)
	if urlErr != nil {
		log.Print("requestURI could not parse, this should not happen: ", urlErr)
		return nil, nil, &GenericErrorResponse{
			Status: "server_error",
			Detail: "requestUri could not parse, this should not happen",
		}
	}

	uuidText := requestUrl.Query().Get("forId")
	jobContainerId, uuidErr := uuid.Parse(uuidText)

	if uuidErr != nil {
		log.Printf("Could not parse forJob parameter %s into a UUID: %s", uuidText, uuidErr)
		return requestUrl, nil, &GenericErrorResponse{"error", "Invalid forJob parameter"}
	}

	return requestUrl, &jobContainerId, nil
}
