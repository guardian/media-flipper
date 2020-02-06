all: reaper
GOFILES := $(wildcard *.go)

reaper: $(GOFILES)
	GOOS=linux CGO_ENABLED=0  go build && cp reaper ../docker

test: $(GOFILES)
	go test

dev: $(GOFILES)
	go build

clean:
	rm -f reaper