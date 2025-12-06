#!/bin/bash
# Setup script for PAW development environment
# Ensures Go 1.24.10 is properly configured

export GOROOT="$HOME/go-installs/go"
export PATH="$GOROOT/bin:$PATH"
export GOPATH="$HOME/go"
export GO111MODULE=on
export PAW_HOME="$HOME/.paw"

echo "Go environment configured for PAW:"
echo "  GOROOT:  $GOROOT"
echo "  GOPATH:  $GOPATH"
echo "  Go version: $(go version)"
echo ""
echo "To use this environment in your current shell, run:"
echo "  source ./setup-go.sh"
