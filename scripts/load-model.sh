#!/bin/bash

if [ "$#" -ne 1 ]; then
	echo "usage: load-model.sh <model-name>"
	exit 1
fi

DEFAULT_URL=http://localhost:8080
V1_BASE_URL=${DEFAULT_URL%"/v1"}
BASE_URL="${V1_BASE_URL:-$DEFAULT_URL}"

MODEL_NAME=$1
echo "loading $MODEL_NAME"

curl --no-progress-meter -X POST "${BASE_URL}/models/load" \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\"}"

echo