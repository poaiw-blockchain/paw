#!/bin/bash
# Install mutation testing tools

set -e

echo "Installing go-mutesting..."
go install example.com/zimmski/go-mutesting/cmd/go-mutesting@latest

echo "Installation complete!"
echo "go-mutesting is now available in your GOPATH/bin"
