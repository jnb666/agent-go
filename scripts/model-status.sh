#!/bin/bash

DEFAULT_URL=http://localhost:8080/v1
BASE_URL="${OPENAI_BASE_URL:-$DEFAULT_URL}"

curl --no-progress-meter "${BASE_URL}/models" | jq '.data[]' | jq -r '{ "model":.id, "alias":.aliases[0], "status":.status.value }'