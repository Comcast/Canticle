#!/bin/sh

cantdir="$GOPATH/src/github.com/Comcast/Canticle"
go get "github.com/Comcast/Canticle/..."
pushd $cantdir
cant genversion
go install "github.com/Comcast/Canticle/cant"
popd
