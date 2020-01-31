package models

import (
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"time"
)

/**
convenience function to perform a mapstructure decode using the customised decode hook below,
to handle UUID and timestamp strings
*/
func CustomisedMapStructureDecode(incoming interface{}, outgoing interface{}) error {
	decoder, setupErr := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructureDecodeHook,
		Result:     outgoing,
	})
	if setupErr != nil {
		return setupErr
	}
	return decoder.Decode(incoming)
}

/**
this custom decode hook will perform a couple of extra conversions:
- if the input type is string and the output is uuid, then it will attempt to parse the uuid and send the error back
up the chain if it can't
- if the input type is string and the output is time, then it will attempt to parse the time as an RFC 3339 timestamp
and send the error back up the chain if it can't.
*/
func mapstructureDecodeHook(inType reflect.Type, outType reflect.Type, value interface{}) (interface{}, error) {
	if inType == reflect.TypeOf("") && outType == reflect.TypeOf(uuid.UUID{}) {
		idvalue, uuidErr := uuid.Parse(value.(string))
		if uuidErr != nil {
			return nil, uuidErr
		} else {
			return idvalue, nil
		}
	} else if inType == reflect.TypeOf("") && outType == reflect.TypeOf(time.Time{}) {
		timeval, timeerr := time.Parse(time.RFC3339, value.(string))
		if timeerr != nil {
			return nil, timeerr
		} else {
			return timeval, nil
		}
	} else {
		return value, nil
	}
}
