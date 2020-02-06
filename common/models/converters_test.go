package models

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestSafeFloat(t *testing.T) {
	//safeFloat should convert an interface{} representing a float
	testvalue := float64(4.8)
	var untyped interface{}
	untyped = testvalue
	result := safeFloat(untyped, 9999)
	if result != testvalue {
		t.Errorf("safeFloat returned wrong value, expected %f got %f", testvalue, result)
	}

	//if an interface is nil then the default value should be returned
	untyped = nil
	nilResult := safeFloat(untyped, 9999)
	if nilResult != 9999 {
		t.Errorf("safeFloat did not return default value when given nil, got %f expected %f", float64(9999), nilResult)
	}

	//if an interface is the wrong type then the default value should be returned
	untyped = "fsfjhsfhjf"
	wrongResult := safeFloat(untyped, 9999)
	if wrongResult != 9999 {
		t.Errorf("safeFloat did not return default value when given string, got %f expected %f", float64(9999), wrongResult)
	}
}

func TestSafeGetString(t *testing.T) {
	//safeGetString should return a string from an interface{}
	var untyped interface{} = "teststring"
	result := safeGetString(untyped)
	if result != "teststring" {
		t.Errorf("safeGetString returned wrong value, expected 'teststring' got '%s'", result)
	}

	//if an interface is nil then an empty string should be returned
	untyped = nil
	nilResult := safeGetString(untyped)
	if nilResult != "" {
		t.Errorf("safeGetString returned wrong value for nil, expected '' got '%s'", result)
	}

	//if an interface is the wrong type then the empty string should be returned
	untyped = 12345
	wrongResult := safeGetString(untyped)
	if wrongResult != "" {
		t.Errorf("safeGetString returned wrong value when given number, expected '' got '%s'", result)
	}
}

func TestSafeGetUUID(t *testing.T) {
	//safeGetUUID should return a uuid from an interface{}
	var untyped interface{} = "716B72B3-E5D4-429E-B389-1DF7A4D0E93F"
	expectedValue := uuid.MustParse("716B72B3-E5D4-429E-B389-1DF7A4D0E93F")
	result := safeGetUUID(untyped)
	if result != expectedValue {
		t.Errorf("safeGetUUID returned wrong value, expected '%s' got '%s'", expectedValue.String(), result.String())
	}

	//if an interface is nil safeGetUUID should return a blank uuid
	untyped = nil
	expected := uuid.UUID{}
	nilResult := safeGetUUID(untyped)
	if nilResult != expected {
		t.Errorf("safeGetUUID returned wrong value for nil, expected '%s' got '%s'", expected.String(), nilResult.String())
	}

	//if an interface is the wrong type safeGetUUID should return a blank uuid
	untyped = 123456
	wrongResult := safeGetUUID(untyped)
	if wrongResult != expected {
		t.Errorf("safeGetUUID returned wrong value for incorrect type, expected '%s' got '%s'", expected.String(), wrongResult.String())
	}

	//if an interface is a string but not a uuid then safeGetUUID should return a blank uuid
	untyped = "fjsfdjkhfsjkhafsfa"
	wrongFormatResult := safeGetUUID(untyped)
	if wrongFormatResult != expected {
		t.Errorf("safeGetUUID returned wrong value for wrong format, expected '%s' got '%s'", expected.String(), wrongFormatResult.String())
	}
}

func TestTimeFromOptionalString(t *testing.T) {
	//TimeFromOptionalString should return a time from an interface{} containing an RFC timestamp
	string := "2020-02-03T04:05:06Z"
	var untyped interface{} = string

	result := TimeFromOptionalString(untyped)
	expected, _ := time.Parse(time.RFC3339, string)
	if result == nil {
		t.Errorf("TimeFromOptionalString returned nil for valid data")
	}
	if result != nil && *result != expected {
		t.Errorf("TimeFromOptionalString returned wrong value, expected '%s' got '%s'", expected.String(), result.String())
	}

	//TimeFromOptionalString should return nil if the incoming value is nil
	untyped = nil
	nilResult := TimeFromOptionalString(untyped)
	if nilResult != nil {
		t.Errorf("TimeFromOptionalString should return nil if given nil, but got '%s'", nilResult.String())
	}

	//TimeFromOptionalString should return nil if the string does not parse
	untyped = "gfddgfdjkhdsgfjkhdgf"
	wrongFormatResult := TimeFromOptionalString(untyped)
	if wrongFormatResult != nil {
		t.Errorf("TimeFromOptionalString should return nil if given the wrong format, but got '%s'", wrongFormatResult.String())
	}

	//TimeFromOptionalString should return nil if the data is the wrong type
	untyped = 4.5678
	wrongTypeResult := TimeFromOptionalString(untyped)
	if wrongFormatResult != nil {
		t.Errorf("TimeFromOptionalString should return nil if given the wrong data type, but got '%s'", wrongTypeResult.String())
	}
}
