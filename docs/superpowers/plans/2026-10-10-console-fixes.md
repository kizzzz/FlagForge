# Console state, key selection and Flag deletion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all three user-reported defects using the approved design.

**Architecture:** Keep the console mounted after its first visit. Load editable Key suggestions through the paginated Go API for the current environment. Delete Flags through a transaction and existing PostgreSQL cascading foreign keys; the list confirms the irreversible deletion and refreshes its current query.

**Tech Stack:** React 18, Ant Design 5, Node built-in tests with jsdom, Go/Gin/GORM, PostgreSQL.

## Task 1: Backend deletion

Files: create `backend/internal/api/flag_delete_integration_test.go`; modify `backend/internal/api/flag_integration_test.go`, `backend/internal/api/flag.go`, `backend/internal/api/router.go`, `backend/internal/service/flag.go`.

- [ ] Write tests asserting DELETE returns an empty 204, detail/history/evaluation subsequently return FLAG_NOT_FOUND, same Key in another environment survives, and Key reuse creates a new UUID. Use setupAPI's isolated PostgreSQL schema. A history DELETE trigger raising an exception must return INTERNAL_ERROR and preserve Flag, rules and history.
- [ ] Run `$env:DATABASE_URL='postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable'; go test -tags=integration ./internal/api -run TestDeleteFlag -count=1`; expect 204 assertions to fail with 405.
- [ ] Implement service transaction using existing `repo.NewFlagRepo(tx).Delete(ctx,id)` and `flagError`; validate handler ID using `flagID`, then `c.Status(204)`. Register `v1.DELETE("/flags/:id", h.Delete)`. Replace the old unsupported-method assertion with a method still unsupported, PUT.
- [ ] Repeat targeted tests; expect PASS. Run all Go tests with integration enabled; expect PASS. Commit only backend paths.

## Task 2: Frontend behavior and regression tests

Files: modify `frontend/src/App.tsx`, `frontend/src/pages/ConsolePage/FlagInputs.tsx`, `frontend/src/pages/FlagListPage/index.tsx`, `frontend/src/api/client.ts`, `frontend/src/mocks/handlers.ts`, `frontend/package.json`, `frontend/package-lock.json`; create focused test harness and tests under `frontend/tests/` and a focused options loader beside FlagInputs if needed.

- [ ] Add jsdom as a development dependency and Node built-in test command. Test real React components and API client, replacing network responses only. Assert console drafts/results/tab survive navigation; Key options include all pages/current environment/disabled Flags/custom typing; obsolete environment responses cannot replace new options. Assert deletion cancellation sends no request, failure preserves row, success refreshes query, duplicate requests are blocked and last-page deletion clamps pagination.
- [ ] Run `npm test`; verify failures expose missing behaviors, rather than harness/import failures.
- [ ] Keep ConsolePage mounted inside Shell after first visit using a hidden wrapper outside Routes. Use an editable AutoComplete with Form environment watch; load page_size=100 pages sequentially, abort superseded requests, refresh on focus and environment change, show loading/empty/error feedback without disabling custom input.
- [ ] Add `flagApi.remove(id, options)` using DELETE and the existing 204 handling. Add a danger deletion button/confirmation including Key/environment and irreversible rule/history removal. Share the existing pending row guard with status updates; on success increment reload; on error report feedback without removing the row. Mirror cascading behavior in the MSW development handler.
- [ ] Run `npm test`, `npm run lint`, `npm run build`; expect PASS. Commit only frontend paths. Review against approved design, then review code quality.

## Task 3: Integration, documentation and actual browser verification

Files: modify `README.md`, approved spec status; create `docs/ai-logs/console-fixes-2026-10-10.md`; add final screenshot evidence under `frontend/evidence/`.

- [ ] Review all changes against the three requirements and approved deletion semantics, then inspect lifecycle, abort handling, duplicate-operation guards and test reliability. Resolve substantive findings before merging.
- [ ] Fast-forward the primary feature branch to the implementation commits. Restart the API from the primary checkout and verify health. Verify console state and Key choices with CUA. Create a unique disposable Flag for actual deletion checks; preserve the five existing records, including dark-mode-toggle.
- [ ] Document route-state retention (refresh resets), Key suggestion behavior, DELETE contract/cascading limitation, frontend test command and genuine red/green/real-browser evidence. Update approved spec status. Commit verified files.
- [ ] Report all three outcomes and test results; retain the existing v1.0.0 release tag. Clean up the isolated worktree only after integration and verification.
