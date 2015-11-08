#!/bin/sh

cantdir="$GOPATH/src/github.com/Comcast/Canticle"
vcstools="golang.org/x/tools/go/vcs"
go get $vcstools
mkdir -p $cantdir
git clone https://github.com/Comcast/Canticle $cantdir
pushd $cantdir
go build cant.go
./cant genversion
go build
mkdir -p "$GOPATH/bin/"
cp cant "$GOPATH/bin/cant"
popd
