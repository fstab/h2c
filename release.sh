#!/bin/bash

set -e

# ----------------------------------------------------------------------
# These are just some notes to myself on how I built the release.
# If you need a release, please get it from https://github.com/fstab/h2c
# ----------------------------------------------------------------------

VERSION=v0.0.10

cat > $GOPATH/src/github.com/fstab/h2c/http2client/version.go <<EOF
package http2client

const (
	VERSION = "$VERSION"
	BUILD_DATE = "`date +%Y-%m-%d`"
)
EOF
go fmt $GOPATH/src/github.com/fstab/h2c/http2client/version.go

docker run -v $GOPATH:/go -t -i fstab/gox bash -c 'GO15VENDOREXPERIMENT=1 gox github.com/fstab/h2c'

BUILD_DIR=$(mktemp -d)
mkdir "$BUILD_DIR/h2c-$VERSION"
cd "$BUILD_DIR/h2c-$VERSION"
mkdir bin LICENSE

# move the files generated with the gox build to the target dir.
mv $GOPATH/h2c_* bin

# copy licenses and make them windows encoded
unix2dos -n $GOPATH/src/github.com/fstab/h2c/LICENSE LICENSE/apache-license.txt
curl https://go.googlecode.com/hg/LICENSE | unix2dos > LICENSE/go-license.txt
curl https://raw.githubusercontent.com/fatih/color/master/LICENSE.md | unix2dos > LICENSE/MIT-license.txt
unix2dos > LICENSE/LICENSE.txt <<EOF
h2c is licensed under the Apache License Version 2.0

3rd party libraries licenses:
  * Go and golang.org/x/net/http2/hpack are licensed under Google's Go license.
  * github.com/fatih/color is licensed under an MIT license.
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
