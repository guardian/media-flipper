all: dockerbuild

.PHONY: dockerbuild

BUILD_NUM = `if [ "${BUILD_NUM}" == "" ]; then echo DEV; else echo ${BUILD_NUM}; fi`
dockerbuild:
	docker build . -t guardianmultimedia/mediaflipper:$(BUILD_NUM)