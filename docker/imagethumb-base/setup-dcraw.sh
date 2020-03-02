#!/bin/sh -e

#also list out the non-dev packages here so they don't get auto-removed at the end
apk add build-base zlib-dev zlib libjpeg-turbo-dev libjpeg-turbo tiff-dev tiff jasper-dev imagemagick mailcap file
mkdir -p /usr/src
cd /usr/src
wget https://downloads.sourceforge.net/project/lcms/lcms/2.9/lcms2-2.9.tar.gz
tar xzf lcms2-2.9.tar.gz
cd lcms2-2.9/
./configure
make && make install
cd ..
rm -rf lcms2-2.9
rm -f lcms2-2.9.tar.gz

wget https://www.dechifro.org/dcraw/archive/dcraw-9.28.0.tar.gz
tar vxzf dcraw-9.28.0.tar.gz
cd dcraw/
gcc -o dcraw -O4 dcraw.c -lm -ljasper -ljpeg -llcms2
./dcraw
mkdir -p /usr/local/bin
mv dcraw /usr/local/bin
cd /usr/src
rm -rf dcraw
rm -f dcraw-9.28.0.tar.gz
apk del build-base zlib-dev libjpeg-turbo-dev tiff-dev
rm -rf /var/cache/apk