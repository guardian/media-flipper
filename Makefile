all: webapp wrapper docker

.PHONY: webapp wrapper docker

clean:
	make -C webapp/ clean
	make -C wrapper/ clean
	make -C docker/ clean
webapp:
	make -C webapp/

wrapper:
	make -C wrapper/

docker:
	make -C docker/