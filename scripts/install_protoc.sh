#!/bin/bash

# Download the specified version of protoc
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.6.0/protoc-3.6.0-linux-x86_32.zip

# Unzip the downloaded archive
unzip protoc-3.6.0-linux-x86_32.zip -d protoc