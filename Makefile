all: webapp wrapper reaper docker

.PHONY: webapp wrapper reaper docker

clean:
	make -C webapp/ clean
	make -C wrapper/ clean
	make -C reaper/ clean
	make -C docker/ clean

test:
	make -C webapp/ test
	make -C wrapper/ test
	make -C reaper/ test

webapp:
	make -C webapp/

wrapper:
	make -C wrapper/

reaper:
	make -C reaper/

docker:
	make -C docker/
