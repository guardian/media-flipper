package helpers

import (
	"encoding/json"
	"errors"
	"net/http"
)

type MockResponseWriterState struct {
	LastWrittenBytes  []byte
	WrittenStatusCode *int
}

type MockResponseWriter struct {
	State *MockResponseWriterState
}

func NewMockResponseWriter() MockResponseWriter {
	return MockResponseWriter{
		State: &MockResponseWriterState{},
	}
}

func (mock MockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (mock MockResponseWriter) Write(msg []byte) (int, error) {
	mock.State.LastWrittenBytes = msg
	return len(msg), nil
}

/*
convenience function to get a string of the last written bytes
*/
func (mock MockResponseWriter) LastWrittenString() string {
	return string(mock.State.LastWrittenBytes)
}

/*
convenience function to parse the last written content from json into a generic map
*/
func (mock MockResponseWriter) LastWrittenJson() (map[string]interface{}, error) {
	var rtn map[string]interface{}

	if len(mock.State.LastWrittenBytes) == 0 {
		return nil, errors.New("No content has yet been written")
	}
	marshalErr := json.Unmarshal(mock.State.LastWrittenBytes, &rtn)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return rtn, nil
}

func (mock MockResponseWriter) WriteHeader(statusCode int) {
	statusCodeCopy := statusCode
	mock.State.WrittenStatusCode = &statusCodeCopy
}

/*
return a boolean indicating whether WriteHader has been called
*/
func (mock MockResponseWriter) HeaderOutput() bool {
	return mock.State.WrittenStatusCode != nil
}
