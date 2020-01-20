all: webapp
GOFILES := $(wildcard *.go)
GOOS := linux

webapp: $(GOFILES)
	GOOS=linux go build && cp webapp ../docker && tar c config/ > ../docker/config.tar && tar c static/ > ../docker/static.tar

dev: $(GOFILES)
	go build

clean:
	rm -f webapp