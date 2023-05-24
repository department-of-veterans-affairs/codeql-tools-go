#!/usr/bin/env bash

set -euo pipefail

# Check if DEBUG is set to true
if [[ "${DEBUG:-}" == "true" ]]; then
  set -x
fi

# Check if .env file exists
if [[ ! -f verify-scans/.env ]]; then
  echo "verify-scans/.env not found!"
  exit 1
fi

# Check if verify-scans.pem exists
if [ ! -f verify-scans.pem ]; then
  echo "verify-scans.pem not found!"
  exit 1
fi

# Check if emass-promotion.pem exists
if [ ! -f emass-promotion.pem ]; then
  echo "emass-promotion.pem not found!"
  exit 1
fi

if [[ "${DEBUG:-}" == "true" ]]; then
  echo "Running Verify Scans Docker container in debug mode..."
  docker run -it --rm --env-file verify-scans/.env \
    -e DEBUG=true \
    -e INPUT_VERIFY_SCANS_PRIVATE_KEY="$(cat verify-scans.pem)" \
    -e INPUT_EMASS_PROMOTION_PRIVATE_KEY="$(cat emass-promotion.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans
else
  echo "Running Verify Scans Docker container..."
  docker run -it --rm --env-file verify-scans/.env \
    -e INPUT_VERIFY_SCANS_PRIVATE_KEY="$(cat verify-scans.pem)" \
    -e INPUT_EMASS_PROMOTION_PRIVATE_KEY="$(cat emass-promotion.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:verify-scans
fi