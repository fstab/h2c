#!/bin/bash

export ORIG_GOPATH="$GOPATH"
export GOPATH="$GOPATH/src/github.com/fstab/h2c/vendor"

rm -rf "$GOPATH"
mkdir "$GOPATH"

# go get github.com/fstab/h2c
go get golang.org/x/net/http2/hpack
go get github.com/fatih/color

find "$GOPATH" -name '.git' | while read dir ; do rm -rf "$dir" ; done
ls "$GOPATH" | grep -v src | while read dir ; do rm -rf "$GOPATH/$dir" ; done
rm -rf "$GOPATH/src/github.com/fstab/h2c"

echo LAST UPDATE: `date` > "$GOPATH/LAST_UPDATE.txt"

export GOPATH="$ORIG_GOPATH"
