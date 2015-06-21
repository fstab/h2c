#!/bin/bash

# ----------------------------------------------------------------------
# These are just some notes to myself on how I built the release.
# If you need a release, please get it from https://github.com/fstab/h2c
# ----------------------------------------------------------------------

docker run -v $GOPATH:/home/go -t -i fstab/gox gox github.com/fstab/h2c

VERSION=v0.0.4

rm -rf /tmp/h2c-$VERSION
mkdir /tmp/h2c-$VERSION
cd /tmp/h2c-$VERSION
mkdir bin LICENSE

mv $GOPATH/h2c_* bin

unix2dos -n $GOPATH/src/github.com/fstab/h2c/LICENSE LICENSE/apache-license.txt
curl https://go.googlecode.com/hg/LICENSE | unix2dos > LICENSE/go-license.txt

unix2dos > LICENSE/LICENSE.txt <<EOF
h2c is licensed under the Apache License Version 2.0
h2c is built using Go and bradfitz/http2. Both are licensed under Google's Go license.
EOF

unix2dos > README.txt <<EOF
h2c is a simple command line client for HTTP/2 servers.

Installation
------------

h2c is a single executable. Find the executable for your platform in the bin directory, and rename it to h2c (or h2c.exe on Windows).

About
-----

See https://github.com/fstab/h2c for more info.
EOF
 
cd ..
zip -r h2c-$VERSION.zip h2c-$VERSION
rm -r h2c-$VERSION
mv h2c-$VERSION.zip $GOPATH/src/github.com/fstab/h2c
