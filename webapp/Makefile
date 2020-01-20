all: webapp
GOFILES := $(wildcard *.go)

webapp: $(GOFILES)
	go build && cp webapp ../docker && cp -r config/ ../docker && cp -r && cp -r static/ ../docker
