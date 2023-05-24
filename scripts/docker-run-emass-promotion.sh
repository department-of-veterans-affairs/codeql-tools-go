#!/usr/bin/env bash

set -euo pipefail

# Check if DEBUG is set to true
if [[ "${DEBUG:-}" == "true" ]]; then
  set -x
fi

# Check if .env file exists
if [[ ! -f emass-promotion/.env ]]; then
  echo "emass-promotion/.env not found!"
  exit 1
fi

# Check if emass-promotion.pem exists
if [[ ! -f emass-promotion.pem ]]; then
  echo "emass-promotion.pem not found!"
  exit 1
fi

if [[ "${DEBUG:-}" == "true" ]]; then
  echo "Running Configure CodeQL Docker container in debug mode..."
  docker run -it --rm --env-file emass-promotion/.env \
    -e DEBUG=true \
    -e INPUT_EMASS_PROMOTION_PRIVATE_KEY="$(cat emass-promotion.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:emass-promotion
else
  echo "Running Configure CodeQL Docker container..."
  docker run -it --rm --env-file emass-promotion/.env \
    -e INPUT_EMASS_PROMOTION_PRIVATE_KEY="$(cat emass-promotion.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:emass-promotion
fi
