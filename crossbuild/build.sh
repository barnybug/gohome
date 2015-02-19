#!/bin/bash

set -ex

rm -rf release; mkdir release

EXE=github.com/barnybug/gohome/processes/gohome
GOOS=linux GOARCH=amd64 goop go build -v -o release/gohome-linux-amd64 $EXE
GOOS=linux GOARCH=386 goop go build -v -o release/gohome-linux-386 $EXE
GOOS=linux GOARCH=arm GOARM=5 goop go build -v -o release/gohome-linux-arm5 $EXE

# compress resulting executables
# workaround for bug (see: https://github.com/pwaller/goupx)
/go/bin/goupx -u=false release/gohome-linux-amd64
upx release/*
chmod -R a+rw release
