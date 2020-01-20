all: wrapper
GOFILES := $(wildcard *.go)

wrapper: $(GOFILES)
	go build && cp wrapper ../docker

clean:
	rm -f wrapper