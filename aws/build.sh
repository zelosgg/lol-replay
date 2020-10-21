#!/bin/bash
set -e

env GOOS=linux GOARCH=amd64 go build -o ./application ../server
zip -r aws-bundle.zip application ../config.json Procfile .ebextensions
