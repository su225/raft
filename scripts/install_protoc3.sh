#!/bin/bash

# Download the specified version of protoc
curl -OL https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip

# Unzip the downloaded archive
unzip protoc-3.2.0-linux-x86_64.zip -d protoc3

# Add the binary directory to the path
export PATH="$PWD/protoc3/bin":$PATH