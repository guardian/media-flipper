FROM guardianmultimedia/imagethumb-base:1
#the base image is build from imagethumb-base also in the parent dir. it's done like this to avoid having to recompile dcraw etc. on every build.
RUN apk --no-cache add imagemagick mailcap

COPY wrapper /opt/mediaflipper/wrapper