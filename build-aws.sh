#!/bin/bash
set -e

env GOOS=linux GOARCH=arm go build -o ./application ./server
zip aws-bundle.zip application config.json
