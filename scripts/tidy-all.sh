#!/bin/bash
set -e

for d in services/* protos; do
  (cd $d && go mod tidy)
done
