#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
export DATABASE_URL="${DATABASE_URL:-postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable}"

run_migrate() {
  local direction="$1"
  local migrate_cli
  if [[ ! -d "$ROOT/backend/migrations" ]]; then
    echo "Migration files are delivered by T1; backend/migrations is not available yet." >&2
    return 1
  fi
  if command -v migrate >/dev/null 2>&1; then
    migrate_cli="$(command -v migrate)"
  else
    migrate_cli="$(go env GOPATH)/bin/migrate"
    if [[ "$(go env GOOS)" == "windows" ]]; then
      migrate_cli="${migrate_cli}.exe"
    fi
    if [[ ! -f "$migrate_cli" ]]; then
      if [[ -f "$ROOT/backend/cmd/migrate/main.go" ]]; then
        (cd "$ROOT/backend" && go run ./cmd/migrate "$direction")
        return
      fi
      echo "Install migrate: cd backend && go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1" >&2
      return 1
    fi
  fi
  if [[ "$direction" == "down" ]]; then
    "$migrate_cli" -path "$ROOT/backend/migrations" -database "$DATABASE_URL" down 1
  else
    "$migrate_cli" -path "$ROOT/backend/migrations" -database "$DATABASE_URL" up
  fi
}
