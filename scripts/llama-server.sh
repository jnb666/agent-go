#!/bin/bash

BIN_DIR="/home/john/llama.cpp/build/bin"
SCRIPT_DIR="/home/john/agent-go/scripts"
export LLAMA_CACHE=none

$BIN_DIR/llama-server --models-preset "${SCRIPT_DIR}/llama-server-config.ini" \
    --fit-target 256 --no-mmproj --models-max 1 \
    --jinja --host 0.0.0.0 --port 8080 

