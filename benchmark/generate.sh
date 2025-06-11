#!/bin/bash

# Script to generate Go protobuf code from proto files

set -e

echo "Generating protobuf code..."

# Create output directory for generated code
mkdir -p generated

# Generate Go code from proto files
protoc \
    --proto_path=proto \
    --go_out=generated \
    --go_opt=paths=source_relative \
    proto/*.proto

echo "Generated protobuf code in generated/ directory"
echo "Files created:"
ls -la generated/ 