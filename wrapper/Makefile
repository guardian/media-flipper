all: wrapper
GOFILES := $(wildcard *.go)

wrapper: $(GOFILES)
	GOOS=linux CGO_ENABLED=0 go build
	cp wrapper ../docker

test: $(GOFILES)
	go test

dev: $(GOFILES)
	go build

clean:
	rm -f wrapper