#!/bin/bash
set -e

cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  echo "No .env found. Copy example.env to .env and edit it first."
  exit 1
fi

if command -v air >/dev/null; then
  exec air
fi

echo "air not installed, falling back to go run (no live reload)."
exec go run .
