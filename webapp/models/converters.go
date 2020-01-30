package models

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"log"
	"reflect"
	"time"
)

/*
Utility functions to handle graceful conversion from untyped interfaces into concrete values
*/

func safeFloat(from interface{}, defaultValue float64) float64 {
	if from == nil {
		return defaultValue
	}
	result, isFloat := from.(float64)
	if !isFloat {
		return defaultValue
	} else {
		return result
	}
}

func safeGetString(from interface{}) string {
	if from == nil {
		return ""
	}
	stringContent, isString := from.(string)
	if !isString {
		log.Printf("WARNING: expected string, got %s", spew.Sdump(from))
		return ""
	}
	return stringContent
}

func safeGetUUID(from interface{}) uuid.UUID {
	stringContent := safeGetString(from)
	if stringContent == "" {
		return uuid.UUID{}
	}
	parsed, parseErr := uuid.Parse(stringContent)
	if parseErr != nil {
		log.Printf("Could not decode UUID from '%s' (jobstepanalysis.go/safeGetUUID)", parseErr)
		return uuid.UUID{}
	}
	return parsed
}

func timeFromOptionalString(maybeStringPtr interface{}) *time.Time {
	if maybeStringPtr == nil {
		return nil
	}

	stringVal, isString := maybeStringPtr.(string)
	if !isString {
		log.Printf("timeFromOptionalString: passed value was %s, expected string", reflect.TypeOf(maybeStringPtr))
		return nil
	}

	//t := time.Time{}
	//marshalErr := t.UnmarshalJSON([]byte(stringVal))	//no idea WHY it fails to unmarshal what it marshalled fine...
	t, marshalErr := time.Parse(time.RFC3339Nano, stringVal)
	if marshalErr != nil {
		log.Printf("ERROR: Could not unmarshal time from string '%s': %s", stringVal, marshalErr)
		return nil
	}
	return &t
}
