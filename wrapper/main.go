package main

import (
	"flag"
	"log"
	"os"
)

/**
we expect the following environment variables to be set:
WRAPPER_MODE={analyse|transcode}
SETTINGS_ID={uuid-string} [only if in transcode mode]
WEBAPP_BASE={url-string}  [url to contact main webapp]
*/
func main() {
	testFilePtr := flag.String("filename", "", "testing option, run on this file")
	flag.Parse()

	switch os.Getenv("WRAPPER_MODE") {
	case "analyse":
		result, err := RunAnalysis(*testFilePtr)

		if err != nil {
			log.Fatal("Could not run analysis: ", err)
		}

		log.Print("Got analysis result: ", result)
	case "transcode":
		log.Fatal("Not yet implemented")
	default:
		log.Fatalf("WRAPPER_MODE '%s' is not recognised", os.Getenv("WRAPPER_MODE"))
	}

}
