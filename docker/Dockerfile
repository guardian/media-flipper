FROM alpine:latest

COPY setup_ffmpeg.sh /tmp
RUN /bin/sh /tmp/setup_ffmpeg.sh
#needed for /etc/mime.types which is used to map file extension to data type
RUN apk add mailcap

COPY wrapper /opt/mediaflipper/wrapper
COPY reaper /opt/mediaflipper/reaper
COPY webapp /opt/mediaflipper/webapp
ADD static.tar /opt/mediaflipper
ADD config.tar /opt/mediaflipper

WORKDIR /opt/mediaflipper
EXPOSE 9000

CMD /opt/mediaflipper/webapp