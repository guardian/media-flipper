package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"sync"
)

func checkFileExists(filepath string) bool {
	_, statErr := os.Stat(filepath)
	if statErr == nil {
		return true
	} else {
		return !os.IsNotExist(statErr)
	}
}

func copyFile(from string, to string) (int64, error) {
	sourceFile, sourceOpenErr := os.Open(from)
	if sourceOpenErr != nil {
		return 0, sourceOpenErr
	}
	defer sourceFile.Close()
	destFile, destOpenErr := os.OpenFile(to, os.O_CREATE|os.O_WRONLY, 0640)
	if destOpenErr != nil {
		return 0, destOpenErr
	}
	defer destFile.Close()

	return io.Copy(destFile, sourceFile)
}

func fileSizeFormatter(size int64) string {
	suffices := []string{"bytes", "Kb", "Mb", "Gb"}

	currentSize := float64(size)
	for i, suffix := range suffices {
		if currentSize < 1024 {
			return fmt.Sprintf("%.1f%s", currentSize, suffix)
		}
		currentSize = float64(size) / math.Pow(1024, float64(i))
	}
	return fmt.Sprintf("%.1fGb", float64(size)/math.Pow(1024, 3))
}

func copierThread(inputCh chan *CopyRequest, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		rec := <-inputCh
		if rec == nil {
			log.Printf("INFO copierThread exiting")
			return
		}

		if !checkFileExists(rec.From) {
			log.Printf("ERROR copierThread %s does not exist to copy from", rec.From)
			continue
		}

		if checkFileExists(rec.To) {
			log.Printf("ERROR copierThread %s already exists, won't over-write", rec.To)
			continue
		}

		mkDirErr := os.MkdirAll(path.Dir(rec.To), 0755)
		if mkDirErr != nil {
			log.Printf("ERROR copierThread can't create directory %s: %s", path.Dir(rec.To), mkDirErr)
			continue
		}

		byteSize, copyErr := copyFile(rec.From, rec.To)
		if copyErr != nil {
			log.Printf("ERROR copierThread can't copy '%s' to '%s': %s", rec.From, rec.To, copyErr)
			continue
		} else {
			log.Printf("INFO copierThread coped %s: %s", rec.To, fileSizeFormatter(byteSize))
		}
	}
}

func AsyncCopier(inputCh chan *CopyRequest, parallelCopies int) chan error {
	errCh := make(chan error, 1)
	modifiedInputCh := make(chan *CopyRequest, cap(inputCh))
	waitGroup := &sync.WaitGroup{}

	for i := 0; i < parallelCopies; i++ {
		go copierThread(modifiedInputCh, waitGroup)
		waitGroup.Add(1)
	}

	go func() {
		for {
			req := <-inputCh
			if req == nil {
				log.Print("INFO AsyncCopier reached end of stream, signalling threads")
				for i := 0; i < parallelCopies; i++ {
					modifiedInputCh <- nil
				}
				log.Print("INFO AsyncCopier waiting...")
				waitGroup.Wait()
				log.Print("INFO AsyncCopier all threads exitied, shutting down")
				errCh <- nil
				return
			}
			modifiedInputCh <- req

			log.Printf("DEBUG AsyncCopier request to copy %s to %s", req.From, req.To)
		}
	}()

	return errCh
}
