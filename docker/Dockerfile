FROM alpine:latest

COPY webapp /opt/mediaflipper/webapp
COPY wrapper /opt/mediaflipper/wrapper
ADD config.tar /opt/mediaflipper
ADD static.tar /opt/mediaflipper

WORKDIR /opt/mediaflipper
EXPOSE 9000

CMD /opt/mediaflipper/webapp