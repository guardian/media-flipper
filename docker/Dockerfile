FROM alpine:latest

COPY setup_ffmpeg.sh /tmp
RUN /bin/sh /tmp/setup_ffmpeg.sh

COPY webapp /opt/mediaflipper/webapp
COPY wrapper /opt/mediaflipper/wrapper
ADD config.tar /opt/mediaflipper
ADD static.tar /opt/mediaflipper

WORKDIR /opt/mediaflipper
EXPOSE 9000

CMD /opt/mediaflipper/webapp