all: webapp wrapper docker

.PHONY: webapp wrapper docker

clean:
	make -C webapp/ clean
	make -C wrapper/ clean
	make -C docker/ clean

test:
	make -C webapp/ test
	make -C wrapper/ test

webapp:
	make -C webapp/

wrapper:
	make -C wrapper/

docker:
	make -C docker/