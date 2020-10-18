#!/usr/bin/env bash
mkdir -p $GOPATH/src/github.com/1lann
ln -s $GOPATH/src/github.com/zelosgg/lol-replay $GOPATH/src/github.com/1lann/lol-replay
cd $GOPATH/src/github.com/1lann/lol-replay
pwd
ls
git status
go get -v -u ./server
go build -o ./app ./server
