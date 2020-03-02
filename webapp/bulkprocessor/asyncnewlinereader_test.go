package bulkprocessor

import (
	"strings"
	"testing"
)

func TestAsyncNewlineReader(t *testing.T) {
	testContent := `this is line1
this is line 2

this is line 4 after blank
this is nothing`

	reader := strings.NewReader(testContent)

	contentChan, errChan := AsyncNewlineReader(reader, nil, 2)

	receivedLines := make([]string, 0)

	func() {
		for {
			select {
			case line := <-contentChan:
				if line == nil {
					return
				}
				receivedLines = append(receivedLines, *line)
			case err := <-errChan:
				t.Error("async returned an error: ", err)
				return
			}
		}
	}()

	expectedData := []string{
		"this is line1",
		"this is line 2",
		"",
		"this is line 4 after blank",
		"this is nothing",
	}

	for i, line := range receivedLines {
		if line != expectedData[i] {
			t.Errorf("async line reader output mismatch on line %d: expected %s, got %s", i, expectedData[i], line)
		}
	}
}
