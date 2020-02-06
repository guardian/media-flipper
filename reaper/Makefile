all: reaper
GOFILES := $(wildcard *.go)

reaper: $(GOFILES)
	GOOS=linux go build && cp reaper ../docker

test: $(GOFILES)
	go test

dev: $(GOFILES)
	go build

clean:
	rm -f reaper