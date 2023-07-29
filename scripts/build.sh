#!/bin/bash

set -euo pipefail

BINARY_NAME=sql2csv

rm -rf bin/${BINARY_NAME}*

GOOS=windows GOARCH=amd64 go build -ldflags "-w" -o bin/${BINARY_NAME}-amd64.exe

GOOS=windows GOARCH=386 go build -ldflags "-w" -o bin/${BINARY_NAME}-386.exe

GOOS=darwin GOARCH=amd64 go build -ldflags "-w" -o bin/${BINARY_NAME}-amd64-darwin

GOOS=linux GOARCH=amd64 go build -ldflags "-w" -o bin/${BINARY_NAME}-amd64-linux

GOOS=linux GOARCH=386 go build -ldflags "-w" -o bin/${BINARY_NAME}-386-linux
