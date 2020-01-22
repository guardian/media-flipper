#!/bin/sh

apk add xz --no-cache
wget https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
xzcat ffmpeg-release-amd64-static.tar.xz | tar x
cp ffmpeg-4.2.2-amd64-static/ffmpeg /usr/bin
cp ffmpeg-4.2.2-amd64-static/ffprobe /usr/bin
cp ffmpeg-4.2.2-amd64-static/qt-faststart /usr/bin
cp -r ffmpeg-4.2.2-amd64-static/model /usr/local/share/
rm -rf ffmpeg-4.2.2-amd64-static
apk del xz --no-cache