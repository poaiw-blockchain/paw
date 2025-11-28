#!/bin/bash
# Install K6 load testing tool

set -e

echo "Installing K6..."

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sudo gpg -k
    sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
    echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
    sudo apt-get update
    sudo apt-get install k6
elif [[ "$OSTYPE" == "darwin"* ]]; then
    brew install k6
else
    echo "Unsupported OS. Please install K6 manually from https://k6.io/docs/getting-started/installation"
    exit 1
fi

echo "K6 installed successfully!"
k6 version
