all: dockerbuild

.PHONY: dockerbuild

DOCKER_VER = `if [ "${BUILD_NUM}" == "" ]; then echo DEV; else echo ${BUILD_NUM}; fi`
dockerbuild:
	docker build . -t guardianmultimedia/mediaflipper:$(DOCKER_VER)

clean:
	rm -f webapp
	rm -f wrapper
	rm -f *.tar