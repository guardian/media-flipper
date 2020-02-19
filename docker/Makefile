all: dockerbuild

.PHONY: dockerbuild

ifndef BUILD_NUM
	DOCKER_VER="DEV"
else
	DOCKER_VER="$(BUILD_NUM)"
endif

dockerbuild:
	docker build . -f masterimage/Dockerfile -t guardianmultimedia/mediaflipper:DEV
	docker build . -f imagethumb/Dockerfile  -t guardianmultimedia/mediaflipper-imagethumb:DEV

clean:
	rm -f webapp
	rm -f wrapper
	rm -f reaper
	rm -f *.tar