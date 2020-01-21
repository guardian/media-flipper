all: dockerbuild

.PHONY: dockerbuild

ifndef BUILD_NUM
	DOCKER_VER="DEV"
else
	DOCKER_VER="$(BUILD_NUM)"
endif

dockerbuild:
	docker build . -t guardianmultimedia/mediaflipper:DEV

clean:
	rm -f webapp
	rm -f wrapper
	rm -f *.tar