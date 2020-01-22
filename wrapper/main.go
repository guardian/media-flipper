package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

func GetMaxRetries() int {
	stringVal := os.Getenv("MAX_RETRIES")
	if stringVal != "" {
		value, err := strconv.ParseInt(stringVal, 10, 16)
		if err != nil {
			log.Fatalf("Invalid value for MAX_RETRIES (not an integer): %s", err)
		}
		return int(value)
	} else {
		return 10 //default value
	}
}

/**
we expect the following environment variables to be set:
WRAPPER_MODE={analyse|transcode}
JOB_ID={uuid-string}
WEBAPP_BASE={url-string}  [url to contact main webapp]
MAX_RETRIES={count}
*/
func main() {
	testFilePtr := flag.String("filename", "", "testing option, run on this file")
	flag.Parse()

	maxTries := GetMaxRetries()
	log.Printf("Max retriues set to %d", maxTries)
	switch os.Getenv("WRAPPER_MODE") {
	case "analyse":
		var filename string
		if os.Getenv("FILE_NAME") != "" {
			filename = os.Getenv("FILE_NAME")
		} else {
			filename = *testFilePtr
		}
		result, err := RunAnalysis(filename)

		if err != nil {
			log.Fatal("Could not run analysis: ", err)
		}

		log.Print("Got analysis result: ", result)
		sendUrl := os.Getenv("WEBAPP_BASE") + "/api/analysis/result?forJob=" + os.Getenv("JOB_ID")
		sendErr := SendToWebapp(sendUrl, result, 0, maxTries)
		if sendErr != nil {
			log.Fatalf("Could not send results to %s: %s", sendUrl, sendErr)
		}
	case "transcode":
		log.Fatal("Not yet implemented")
	default:
		log.Fatalf("WRAPPER_MODE '%s' is not recognised", os.Getenv("WRAPPER_MODE"))
	}

}
