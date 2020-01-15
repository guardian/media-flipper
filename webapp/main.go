package main

import (
	"log"
	"net/http"
)

type MyHttpApp struct {
	index  IndexHandler
	static StaticFilesHandler
}

func main() {
	var app MyHttpApp

	app.index.filePath = "public/index.html"
	app.index.contentType = "text/html"
	app.index.exactMatchPath = "/"
	app.static.basePath = "public"
	app.static.uriTrim = 2

	http.Handle("/default", http.NotFoundHandler())
	http.Handle("/", app.index)
	http.Handle("/static/", app.static)

	log.Printf("Starting server on port 9000")
	startServerErr := http.ListenAndServe(":9000", nil)

	if startServerErr != nil {
		log.Fatal(startServerErr)
	}
}
