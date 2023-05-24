#!/usr/bin/env bash

set -euo pipefail

# Check if DEBUG is set to true
if [[ "${DEBUG:-}" == "true" ]]; then
  set -x
fi

# Check if .env file exists
if [[ ! -f configure-codeql/.env ]]; then
  echo "configure-codeql/.env not found!"
  exit 1
fi

# Check if configure-codeql.pem exists
if [[ ! -f configure-codeql.pem ]]; then
  echo "configure-codeql.pem not found!"
  exit 1
fi

# Check if verify-scans.pem exists
if [[ ! -f verify-scans.pem ]]; then
  echo "verify-scans.pem not found!"
  exit 1
fi

if [[ "${DEBUG:-}" == "true" ]]; then
  echo "Running Configure CodeQL Docker container in debug mode..."
  docker run -it --rm --env-file configure-codeql/.env \
    -e DEBUG=true \
    -e INPUT_CONFIGURE_CODEQL_PRIVATE_KEY="$(cat configure-codeql.pem)" \
    -e INPUT_VERIFY_SCANS_PRIVATE_KEY="$(cat verify-scans.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:configure-codeql
else
  echo "Running Configure CodeQL Docker container..."
  docker run -it --rm --env-file configure-codeql/.env \
    -e INPUT_CONFIGURE_CODEQL_PRIVATE_KEY="$(cat configure-codeql.pem)" \
    -e INPUT_VERIFY_SCANS_PRIVATE_KEY="$(cat verify-scans.pem)" \
    ghcr.io/department-of-veterans-affairs/codeql-tools:configure-codeql
fi
