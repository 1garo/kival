#!/bin/bash

RUN_TEST="${1:-}"
if [ -z "$RUN_TEST" ]; then
  go test ./... \
    -tags=integration \
    -v
else
  go test ./... \
    -tags=integration \
    -run "$1" \
    -v
fi
