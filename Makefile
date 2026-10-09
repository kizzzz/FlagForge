SHELL := bash

.PHONY: dev migrate-up migrate-down seed test smoke down

# Install the PostgreSQL migration CLI:
# cd backend && go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
# Ensure $(go env GOPATH)/bin is on PATH; scripts also look in that directory.
dev:
	bash scripts/dev.sh

migrate-up:
	bash scripts/migrate-up.sh

migrate-down:
	bash scripts/migrate-down.sh

seed:
	bash scripts/seed.sh

test:
	bash scripts/test.sh

smoke:
	bash scripts/smoke.sh

down:
	bash scripts/down.sh
