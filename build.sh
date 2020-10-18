#!/usr/bin/env bash
mkdir -p $GOPATH/src/github.com/1lann
ln -s $GOPATH/src/github.com/zelosgg/lol-replay $GOPATH/src/github.com/1lann/lol-replay
cd $GOPATH/src/github.com/1lann/lol-replay
git checkout master
go get -v -u ./...
go build -o ./app ./server
