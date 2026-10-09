#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
if [[ ! -f "$ROOT/backend/seed/main.go" ]]; then
  echo "Seed data is delivered by T1; backend/seed/main.go is not available yet." >&2
  exit 1
fi
cd "$ROOT/backend"
go run ./seed
