#!/bin/bash
# Wrapper script to run commands with Go 1.24.10 available
# Usage: ./run-with-go.sh go test ./...
#        ./run-with-go.sh go build ./cmd/...

export GOROOT="$HOME/go-installs/go"
export PATH="$GOROOT/bin:$PATH"
export GOPATH="$HOME/go"
export GO111MODULE=on
export PAW_HOME="$HOME/.paw"

exec "$@"
