package main

import (
	"github.com/guardian/mediaflipper/common/helpers"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type IndexHandler struct {
	handler http.Handler

	filePath       string
	contentType    string
	exactMatchPath string
}

func (h IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestUrl, _ := url.ParseRequestURI(r.RequestURI)

	if h.exactMatchPath != "" && requestUrl.Path != h.exactMatchPath {
		log.Printf("Requested URL %s did not match exactMatchPath %s for this controller", requestUrl.Path, h.exactMatchPath)
		w.WriteHeader(404)
		return
	}

	if strings.HasPrefix(requestUrl.Path, "/api") {
		log.Printf("Access for invalid API path %s fell through to html handler, returning json 404", requestUrl.Path)
		helpers.WriteJsonContent(helpers.GenericErrorResponse{
			Status: "not_found",
			Detail: "invalid api endpoint",
		}, w, 404)
		return
	}

	f, openErr := os.Open(h.filePath)

	if openErr != nil {
		log.Printf("Could not get index.html: %s", openErr)
		w.WriteHeader(500)
		return
	}

	statInfo, statErr := os.Stat(h.filePath)
	if statErr != nil {
		log.Printf("Could not get index.html: %s", openErr)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Length", strconv.FormatInt(statInfo.Size(), 10))
	w.Header().Add("Content-Type", h.contentType)
	w.WriteHeader(200)

	_, err := io.Copy(w, f)

	if err != nil {
		log.Print("Could not output frontend: ", err)
	}
}
