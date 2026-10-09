#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
cd "$ROOT"

docker compose up -d
ready=false
for ((attempt = 0; attempt < 30; attempt++)); do
  if docker compose exec -T postgres pg_isready -U flagforge -d flagforge >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo "PostgreSQL did not become ready; inspect docker compose logs postgres." >&2
  exit 1
fi

if [[ -d backend/migrations ]]; then
  bash scripts/migrate-up.sh
fi
if [[ -f backend/seed/main.go ]]; then
  bash scripts/seed.sh
fi

server_binary="$ROOT/backend/bin/server"
if [[ "$(go env GOOS)" == "windows" ]]; then
  server_binary="${server_binary}.exe"
fi
mkdir -p backend/bin
(cd backend && go build -o "$server_binary" ./cmd/server)
"$server_binary" &
backend_pid=$!
cleanup() {
  kill "$backend_pid" 2>/dev/null || true
  wait "$backend_pid" 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ ! -d frontend/node_modules ]]; then
  (cd frontend && npm ci)
fi
echo "Backend http://localhost:8787/healthz; frontend http://localhost:5173"
(cd frontend && npm run dev)
