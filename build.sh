#!/usr/bin/env bash
mkdir /var/data/recordings
go get -v -u ./...
go build -o ./app ./server

