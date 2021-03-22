package main

import (
	"flag"
	"github.com/google/uuid"
	"log"
	"os"
)

/**
this app reads the job history to determine the proxy and thumbnail for each record then copies the proxies to a
new directory tree matching the original media.
this is to make it easier to use the transcodes in ArchiveHunter and VaultDoor
*/

func checkOutputDirectory(dirpath string) bool {
	statInfo, statErr := os.Stat(dirpath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			log.Printf("ERROR the output path '%s' does not exist, can't continue", dirpath)
			return false
		}
		log.Printf("ERROR can't check path '%s': %s", dirpath, statErr)
		return false
	}

	if !statInfo.IsDir() {
		log.Printf("ERROR the output path '%s' is not a directory, can't continue", dirpath)
		return false
	}
	return true
}

func main() {
	limitToBulkIdPtr := flag.String("bulk", "", "if set, only copy items from the given bulk")
	outputPath := flag.String("output", "", "path to output files to")
	baseUrl := flag.String("url", "http://localhost:9000", "location of the mediaflipper webapp")
	pathStripCount := flag.Int("pathstrip", 2, "remove this many path segments from the original media path when calculating destination media path")
	pageSize := flag.Int("pagesize", 100, "how many items to grab at once")
	parallelCopies := flag.Int("parallel", 4, "how many copies to perform in parallel")
	flag.Parse()

	if *outputPath == "" {
		log.Fatal("You must specify an output path using '--output'")
	}

	if !checkOutputDirectory(*outputPath) {
		log.Fatal("Invalid output directory")
	}

	var limitToBulkId *uuid.UUID
	if *limitToBulkIdPtr != "" {
		uid, uidErr := uuid.Parse(*limitToBulkIdPtr)
		if uidErr != nil {
			log.Fatal("If you specify a value for -bulk it must be a valid UUID. Error was ", uidErr)
		}
		limitToBulkId = &uid
	}

	allJobsCh, jobScanErrCh := AsyncScanJobs(*baseUrl, *pageSize)
	filteredJobsCh := AsyncFilterBulkId(allJobsCh, limitToBulkId, *pageSize)
	copyReqCh, reqbuilderErrCh := AsyncCopyRequestBuilder(filteredJobsCh, *baseUrl, *outputPath, *pathStripCount, *pageSize)
	copyErrCh := AsyncCopier(copyReqCh, *parallelCopies)

	func() {
		select {
		case err := <-jobScanErrCh:
			log.Print("ERROR main got error from job scanner: ", err)
			return
		case err := <-reqbuilderErrCh:
			log.Print("ERROR main got error from request builder: ", err)
			return
		case err := <-copyErrCh:
			if err == nil {
				log.Print("INFO Completed")
				return
			}
			log.Print("ERROR main got error from copier: ", err)
			return
		}
	}()

	log.Print("All done")
}
