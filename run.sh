#!/bin/bash
echo "cleanup"
#keep data
rm -rf bin/log bin/resources bin/.env bin/vpr
echo "copy resources"
mkdir -p bin
mkdir -p bin/resources
cd go
cp -R ../resources ../bin
cp .env ../bin
go version
#echo "unit tests"
#go test -v ./... -cover -coverprofile=coverage.out
echo
echo "building"
export BUILD_VERSION=0.0.2
envsubst <../bin/.env | tee ../bin/.env
time=$(date)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.BuildTime=$time' -X 'main.BuildVersion=$BUILD_VERSION'" -o ../bin/vpr .
cd ../bin
# Set all ENV vars for the server to run
export $(grep -v '^#' .env | xargs)
echo
sleep 1
./vpr --debug
