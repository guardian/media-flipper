all: webapp
GOFILES := $(wildcard *.go)
GOOS := linux

webapp: $(GOFILES)
	GOOS=linux CGO_ENABLED=0 go build
	cp webapp ../docker
	tar c config/ > ../docker/config.tar
	tar c static/ > ../docker/static.tar

dev: $(GOFILES)
	go build

test: $(GOFILES)
	go test ./...

clean:
	rm -f webapp
	find . -iname \*.out -delete