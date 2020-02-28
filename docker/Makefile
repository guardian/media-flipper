all: dockerbuild

.PHONY: dockerbuild clean push

ifndef BUILD_NUM
	DOCKER_VER="DEV"
else
	DOCKER_VER="$(BUILD_NUM)"
endif

dockerbuild:
	docker build . -f masterimage/Dockerfile -t guardianmultimedia/mediaflipper:DEV
	docker build . -f imagethumb/Dockerfile  -t guardianmultimedia/mediaflipper-imagethumb:DEV
	docker build . -f awsupload/Dockerfile -t guardianmultimedia/mediaflipper-aws:DEV

clean:
	rm -f webapp
	rm -f wrapper
	rm -f reaper
	rm -f *.tar

push:
	docker push guardianmultimedia/mediaflipper:DEV
	docker push guardianmultimedia/mediaflipper-imagethumb:DEV
	docker push guardianmultimedia/mediaflipper-aws:DEV