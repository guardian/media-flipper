package main

import (
	"log"
	"os"
)

func checkFileExists(filepath string) bool {
	_, statErr := os.Stat(filepath)
	if statErr==nil {
		return true
	} else {
		return !os.IsNotExist(statErr)
	}
}

func AsyncCopier(inputCh chan *CopyRequest) chan error {
	errCh := make(chan error, 1)

	go func() {
		for {
			req := <- inputCh
			if req==nil {
				log.Print("INFO AsyncCopier reached end of stream, terminating")
				errCh <- nil
				return
			}

			log.Printf("DEBUG AsyncCopier request to copy %s to %s", req.From, req.To)
		}
	}()

	return errCh
}
