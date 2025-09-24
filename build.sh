#!/bin/bash
echo "cleanup"
rm -rf bin
echo "copy resources"
mkdir -p bin
cd go
cp -R resources ../bin
cp .env ../bin
go version
echo "unit tests"
go test -v ./... -cover -coverprofile=coverage.out
echo
echo "Overall Coverage"
go tool cover -func coverage.out | grep total: 
echo
echo "building"
export BUILD_VERSION=0.0.2
envsubst <../bin/.env | tee ../bin/.env
time=$(date)
CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.BuildTime=$time' -X 'main.BuildVersion=$BUILD_VERSION'" -o ../bin/vpr-exporter .
