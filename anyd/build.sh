#!/usr/bin/env bash

if [ ! -f "build.sh" ]; then
echo 'build must be run within its container folder' 1>&2
exit 1
fi

CURDIR=`pwd`
OLDGOPATH="$GOPATH"
export GOPATH="$OLDGOPATH:$CURDIR"

gofmt -w src

# go build ./src/examples/testtoml.go

# go install service
# go build -o bin/ngxfmd ${CURDIR}/src/service/ngxfm
go build -o bin/ngxfmd "service/ngxfm"
go build -o bin/qktool "service/qianke/qkdb"

if [ ! -f ./bin/ngxfmd ]; then
echo 'do not exist ./bin/ngxfmd file.'
exit 1
fi

# mv ./bin/service ./bin/ngxfmd

export GOPATH="$OLDGOPATH"

echo 'finished'