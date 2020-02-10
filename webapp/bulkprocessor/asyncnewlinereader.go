package bulkprocessor

import (
	"bufio"
	"io"
)

/**
read line-by-line from src until EOF and push each result as a string pointer to the output channel.
on completion, a nil is pushed to the output channel
on error, a single error is pushed to the error channel
*/
func AsyncNewlineReader(src io.Reader, bufferSize int) (chan *string, chan error) {
	scanner := bufio.NewScanner(src)
	scanner.Split(bufio.ScanLines)

	outputChan := make(chan *string, bufferSize)
	errorChan := make(chan error)

	go func() {
		for {
			moreContent := scanner.Scan()
			if moreContent {
				retrievedString := scanner.Text() //doc says this is newly-allocated so we can safely hand it off here
				outputChan <- &retrievedString
			} else {
				err := scanner.Err()
				if err != nil {
					errorChan <- err
					return
				} else {
					outputChan <- nil
					return
				}
			}
		}
	}()

	return outputChan, errorChan
}
