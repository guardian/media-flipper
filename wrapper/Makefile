all: wrapper
GOFILES := $(wildcard *.go)

wrapper: $(GOFILES)
	GOOS=linux go build && cp wrapper ../docker

dev: $(GOFILES)
	go build

clean:
	rm -f wrapper