# Implementation Plan

## Purpose

Design and prototype a lightweight, locally‑hosted orchestrator that coordinates 3‑5 Sourcegraph Amp coding agents so that they can deliver features faster and with higher code quality than a single diligent agent working alone.

## Problem Statement

A single agent can only work serially: it writes code, runs tests, waits, and repeats. Teams of engineers succeed via parallelism (multiple streams of work) and peer review. We want to replicate those advantages while controlling merge conflicts and ensuring tests remain green.

## 0 · Bootstrap & Project Skeleton

- [x] **Step 0-1: Initialise Go module & repo layout**

  - **Task**: create baseline folders, `.gitignore`, simple `Makefile`.
  - **Why**: establishes consistent imports & build targets.
  - **Files** (4)
    - `go.mod`
    - `.gitignore`
    - `Makefile` – adds `build / test / run / lint`
    - `README.md`
  - **Tests**: none yet (scaffolding only).
  - **User Instructions**
    1. `go mod tidy && make build` → binary should compile.
    2. `make lint` (will noop until govet added in 0-2).

- [x] **Step 0-2: Configuration scaffolding**

  - **Task**: add `config.sample.yaml`, `internal/config` loader using Viper + validation.
  - **Files** (4)
    - `config.sample.yaml`
    - `internal/config/config.go`
    - `internal/config/config_test.go` – loads sample & asserts defaults.
    - `cmd/daemon/main.go` (stub)
  - **Tests**: `go test ./internal/config` expects unmarshalled struct to equal known constants.
  - **User Instructions**
    1. `cp config.sample.yaml ~/.config/orchestrator/config.yaml`
    2. `go test ./internal/config` → PASS.

- [x] **Step 0-3: ci.sh & post-receive hook template**
  - **Task**: copy ci script; Go helper installs hook into bare repo.
  - **Files** (4)
    - `ci.sh` (executable bit)
    - `scripts/install_hook.go`
    - `scripts/install_hook_test.go` – uses a temp bare repo & asserts hook written.
    - `repo.git/hooks/post-receive` (template text)
  - **Tests**: run helper in -race temp dir; inspect hook contents.
  - **User Instructions**
    1. `mkdir demo && cd demo && git init --bare repo.git`
    2. `go run ../../scripts/install_hook.go --repo ./repo.git`
    3. `cat repo.git/hooks/post-receive` → should call `ci.sh`.

---

## 1 · Scheduler Core (Sprint 1)

- [x] **Step 1-1: Ticket struct & YAML loader**

  - **Files** (3): `internal/ticket/ticket.go`, `..._test.go`, `examples/avatar.yaml`.
  - **Tests**: valid YAML parses; missing field fails with custom error.
  - **User Instructions**
    - `go test ./internal/ticket`
    - `orchestrator validate examples/avatar.yaml` (CLI to be added next step) should print ✅.

- [x] **Step 1-2: Priority queue implementation**

  - **Files** (3): `internal/queue/queue.go`, `..._test.go`, `internal/queue/heap.go`.
  - **Tests**: push 3 priorities, pop yields expected order.
  - **User Instructions**
    - `go test ./internal/queue`

- [x] **Step 1-3: Backlog watcher (fsnotify + ticker)**
  - **Files** (4): `internal/watch/watch.go`, `..._test.go`, `cmd/daemon/main.go` (wire), `cmd/cli/main.go` (add `enqueue` & `validate` cmds).
  - **Tests**: temp dir watcher fires event; ticker fallback fires at least once.
  - **Manual Check**
    1. `make run` (daemon).
    2. `orchestrator enqueue examples/avatar.yaml`.
    3. Daemon log prints “Enqueued ticket #123”.

---

## 2 · Worker Management MVP

- [x] **Step 2-1: Git utility helpers**

  - **Files** (4): `pkg/gitutils/git.go`, `..._test.go`, `internal/errors.go`, `Makefile` (+ `go vet`).
  - **Tests**: create bare repo in tempdir, `AddWorktree` returns expected path; `CommitFile` pushes commit count = 1.
  - **Manual Check**
    - `go test ./pkg/gitutils`
    - Inspect `tmp/repo.git/refs/heads/...` count.

- [x] **Step 2-2: Amp-Worker stub**
  - **Files** (4): `internal/worker/worker.go`, `..._test.go`, `cmd/daemon/main.go` (invoke), `docs/SPRINT1_DEMO.md`.
  - **Tests**: worker creates branch `agent-X/feat-id`; CI script triggered (mock).
  - **Manual Check**
    1. `make run` (daemon still up).
    2. Enqueue ticket; `git --git-dir repo.git branch -a` shows `agent-1/...`.

---

## 3 · Local CI Integration

- [x] **Step 3-1: ci.sh status JSON**

  - **Files** (3): `ci.sh`, `internal/ci/status.go`, `internal/ci/status_test.go`.
  - **Tests**: run `ci.sh` in temp worktree; file `ci-status/<hash>.json` exists & contains `"status":"PASS"`.
  - **Manual Check**
    - Push commit and `ls repo.git/ci-status`.

- [x] **Step 3-2: Hook installer v2**
  - **Files** (2): `scripts/install_hook.go`, `scripts/install_hook_test.go` (+ new assertions).
  - **Tests**: verify `$1` param forwarded, status dir created by hook.
  - **Manual Check**
    - Re-install hook; push failing commit → new JSON `"FAIL"` appears.

---

## 4 · Bubble Tea TUI α

- [ ] **Step 4-1: Unix-socket JSON event bus**

  - **Files** (4): `internal/ipc/ipc.go`, `..._test.go`, `cmd/daemon/main.go` (publish), `cmd/cli/tui.go` (subscribe).
  - **Tests**: client subscribes; receives ≥1 `QueueEvent` within timeout.
  - **Manual Check**
    - Run daemon, then `nc -U ~/.orchestrator.sock` → raw JSON streams.

- [ ] **Step 4-2: Render Tickets & Agents panels**

  - **Files** (3): `cmd/cli/tui.go`, `cmd/cli/tui_model.go`, `cmd/cli/tui_view.go`.
  - **Tests**: none (UI difficult to unit test); instead integration check next.
  - **Manual Check**
    - `orchestrator tui` shows two panels; adding ticket updates list in <1 s.

- [ ] **Step 4-3: Key-bindings c/s/q**
  - **Files** (3): `cmd/cli/tui_update.go`, `internal/ipc/handlers.go`, `internal/ipc/handlers_test.go`.
  - **Tests**: send `ContinueMsg`; scheduler stub returns success channel; state == approved.
  - **Manual Check**
    - In TUI, highlight agent row, press `c` → row turns green; press `s` → turns yellow.

---

## 5 · Locking Mechanism

- [ ] **Step 5-1: Lock file data model & parser**

  - **Files** (3): `internal/locks/file.go`, `..._test.go`, `examples/locks.json`.
  - **Tests**: round-trip read/write preserves map; missing file returns empty map.
  - **Manual Check**
    - `orchestrator locks list` prints empty, then after edit prints paths.

- [ ] **Step 5-2: Scheduler lock acquisition**

  - **Files** (3): `internal/scheduler/locks.go`, `..._test.go`, `internal/scheduler/scheduler.go` (update).
  - **Tests**: two goroutines compete; first wins & second retries as specified.
  - **Manual Check**
    - Enqueue two tickets with same `locks:`; second ticket remains “WAITING”.

- [ ] **Step 5-3: Auto-unlock timeout & stale detection**
  - **Files** (3): `internal/locks/timeout.go`, `..._test.go`, `cmd/daemon/main.go` (enable).
  - **Tests**: fake agent heartbeat stops; unlock commit created after grace window.
  - **Manual Check**
    - Kill worker process; watch UI row turn red then disappear.

---

## 6 · Reviewer Agent

- [ ] **Step 6-1: Reviewer process launcher**
  - **Files** (4): `internal/reviewer/reviewer.go`, `..._test.go`, `internal/worker/worker.go`, `cmd/cli/tui_view.go`.
  - **Tests**: reviewer stub returns JSON comment list; UI parses count.
  - **Manual Check**
    - After worker pushes, TUI shows 📝 icon & comment count.

---

## 7 · Merger Automation

- [ ] **Step 7-1: Fast-forward rebase & smoke tests**
  - **Files** (4): `internal/merger/merger.go`, `..._test.go`, `internal/scheduler/scheduler.go`, `ci.sh` (add `--quick`).
  - **Tests**: given separate main & branch commits, merger rebases cleanly, runs smoke test stub.
  - **Manual Check**
    - Press `c` on green agent; branch disappears, `git log main` shows feature commit.

---

## 8 · Metrics & Experiment Harness

- [ ] **Step 8-1: Metrics collector & CSV writer**

  - **Files** (3): `internal/metrics/collector.go`, `..._test.go`, `cmd/daemon/main.go`.
  - **Tests**: enqueue→merge path produces one CSV row with non-zero duration.
  - **Manual Check**
    - After a few tickets, open `metrics/*.csv` in spreadsheet.

- [ ] **Step 8-2: Jupyter notebook placeholder**
  - **Files** (1): `metrics/analysis.ipynb`.
  - **Manual Check**
    - Open notebook; run cell that loads CSV & prints head.

---

## 9 · Release & Packaging

- [ ] **Step 9-1: goreleaser config & dist output**

  - **Files** (2): `.goreleaser.yaml`, `Makefile` (`release`).
  - **Tests**: none (binary packaging).
  - **Manual Check**
    - `make release` → tarball appears in `dist/`.

- [ ] **Step 9-2: systemd service unit**
  - **Files** (2): `examples/orchestrator.service`, `docs/quickstart.md`.
  - **Tests**: none.
  - **Manual Check**
    - `systemctl --user start orchestrator` then `systemctl --user status orchestrator` shows active.
