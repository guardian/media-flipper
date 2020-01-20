all: webapp wrapper docker

.PHONY: webapp wrapper docker

webapp:
	make -C webapp/

wrapper:
	make -C wrapper/

docker:
	make -C docker/