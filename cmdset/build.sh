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

go install service

if [ ! -f ./bin/service ]; then
echo 'do not exist ./bin/service file.'
exit 1
fi

mv ./bin/service ./bin/cmdServer

export GOPATH="$OLDGOPATH"

echo 'finished'