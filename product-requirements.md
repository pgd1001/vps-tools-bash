#PRD (Product Requirements Document)

## Purpose

Replace a collection of maintenance bash scripts with a single, maintainable, secure, and attractive terminal application that supports both scripted (non-interactive) and interactive TUI workflows for managing virtual Linux servers.

## Stakeholders

* Primary: You (owner/operator), vps-tools clients
* Secondary: sysadmins, engineers who will maintain the codebase

## Key features (MVP)

1. **Inventory management**

   * Add / edit / remove servers.
   * Group by tags.
   * Import current bash inventories automatically.

2. **SSH orchestration**

   * Run commands across multiple servers concurrently.
   * Capture stdout/stderr, exit codes, timestamps.
   * Optional interactive shell per server (proxying terminal).

3. **Job scheduling & retry**

   * Queue jobs with concurrency limits and retry policies.

4. **Audit & logging**

   * Append-only logs for audit, exportable.

5. **TUI**

   * Interactive terminal app with list / details / job runner / log viewer (Bubble Tea).
   * Aesthetic styling (Lip Gloss).

6. **CLI**

   * Scriptable non-interactive commands (Cobra) that can be used in CI or cron.

7. **Security**

   * Prefer SSH agent; encrypted config option; explicit confirmation for destructive actions.

## Non-functional requirements

* Performance: handle parallel operations on 200 servers with configurable concurrency.
* Single static binary delivery for major platforms.
* Code style: gofmt / golangci-lint enforced.
* Test coverage: unit tests for core logic; integration tests for SSH wrapper (mockable).
* Maintainability: modular package structure, clear docs, README, CONTRIBUTING, codegen for CLI with Cobra.

## Acceptance criteria (MVP)

* CLI commands `vps-tools status --targets=tag:web` and `vps-tools run --targets=all -- cmd` execute and return structured JSON with job results.
* TUI lists servers, allows selection, runs a command and displays live streaming stdout.
* No feature requires `sh -c` for normal operations; when shelling out is used fallback documented and gated.
* All actions are logged to audit store with timestamp, identity (local user), command and server id.
* Cross-compiled Linux amd64 binary produced via CI pipeline.

---

# Implementation checklist & tasks for the AI coding tool

Deliver these items as separate PRs (one PR per numbered task):

**PR-000: repo skeleton**

* Create module `module github.com/<you>/vps-tools`
* Basic `Makefile` or `magefile` with `build`, `test`, `fmt`, `lint`, `cross-compile`.
* CI: GitHub Actions skeleton that runs `go test`, `gofmt`, `golangci-lint`.

**PR-001: domain & store**

* Implement `internal/server` and `internal/store` (BoltDB or SQLite choice).
* Provide migration helper: parse inventory from existing script outputs (JSON/YAML) into store.

**PR-002: SSH wrapper**

* Implement `internal/ssh` using `golang.org/x/crypto/ssh` with:

  * Support for agent, private key paths, ProxyJump (support bastion).
  * Methods: `RunCommand(server Server, command string, timeout time.Duration) (JobResult, error)`, `StartInteractiveShell(...)`.
* Provide unit tests and an integration test scaffold (mock SSH server).

**PR-003: CLI scaffolding**

* Add Cobra with top-level commands:

  * `vps-tools inventory list`
  * `vps-tools run --targets=... --command="..."`
  * `vps-tools export --format=json`
* Support `--output=json` for machine-readable outputs.

**PR-004: Job engine**

* Implement `internal/jobs` with concurrency pool, retries, timeouts and result collection.
* Add metrics (simple counters).

**PR-005: TUI MVP**

* Implement Bubble Tea app with:

  * Server list (table)
  * Job runner modal (run command on selected servers)
  * Live log view (stream stdout)
* Use Lip Gloss for a clean layout; Bubbles for selectable list and text input.
* Add keyboard shortcuts and accessible help overlay.

**PR-006: Security & secrets**

* Add keyring integration (optional) or encrypted config file support.
* Validate inputs; disallow untrusted interpolation into remote commands.

**PR-007: Tests & docs**

* Unit tests for domain packages.
* Integration test examples for SSH.
* README with install and migration instructions, example config.

**PR-008: Packaging & release**

* Add GitHub Actions release workflow (builds and uploads binaries).
* Provide homebrew formula / install script example.

Each PR must include: code, tests, README updates, and at least one example usage.

---

# UX notes (TUI specifics)

* Use keyboard-first design: `j/k` to move, `Enter` to open, `r` to run, `l` to tail logs, `:` to open command palette.
* Present job results in a table with columns: Server, Exit, Duration, Last line of stdout, Tags.
* Provide filters: by tag, by name, by status.
* For long outputs, offer a paging view with search and colourised stderr vs stdout.

---

# Security checklist (minimum)

* Prefer SSH libraries over `os/exec("ssh ...")`. If shelling out used as fallback, construct args safely (no `sh -c`).
* Use SSH agent where possible. Do not persist plaintext passwords.
* Encrypt local config if it contains private keys; recommend OS keyring.
* Require explicit user prompt/confirmation for destructive tasks.
* Audit logs are tamper-resistant: append-only file with rolling hash (optional) or store in secured location.

Reference for SSH lib and Cobra: `golang.org/x/crypto/ssh` and `spf13/cobra`. ([Go Packages][4])
Reference for Bubble Tea ecosystem and examples: Bubble Tea repo and Lip Gloss examples. ([GitHub][1])

---

# Example acceptance tests (machine-readable)

1. `vps-tools inventory import --from=old-inventory.json` → returns `200 imported`.
2. `vps-tools run --targets=tag:db --command="uname -a" --output=json` → returns valid JSON array with one element per server containing `server_id`, `exit_code`, `stdout`, `stderr`.
3. In TUI: open `vps-tools tui`, select two servers, press `r`, type `uptime` → both jobs display live output and then green success indicator.

---

# Notes & alternatives considered

* Alternatives: Python Textual / Rich, Rust `ratatui`/`tui`, Node `ink`. They are capable, but:

  * Python solutions are easier for quick hacks but are slower and harder to distribute as single binaries.
  * Rust yields excellent performance/safety but longer dev ramp and smaller ecosystem for TUI components compared with Bubble Tea’s ecosystem in Go.
    If you need, I can produce a short pros/cons table comparing these alternatives. ([DEV Community][5])

---

# Final deliverable for the AI coding tool (copy-ready)

Below is a concise instruction block you can paste to an AI coding tool or developer as the implementation brief:

```
Project: vps-tools — Terminal server maintenance app (Go + Bubble Tea)

Stack:
- Go 1.21+ (module mode)
- TUI: Bubble Tea + Bubbles + Lip Gloss
- CLI: spf13/cobra
- SSH: golang.org/x/crypto/ssh (agent support, bastion/proxyjump)
- Store: BoltDB or SQLite (choose BoltDB for simple key-value)
- CI: GitHub Actions (build/test/cross-compile)
- Lint: gofmt + golangci-lint

Deliverables (PR sequence):
PR-000 repo skeleton + CI
PR-001 domain & store models + migration helper
PR-002 SSH wrapper library (agent, key, ProxyJump)
PR-003 CLI scaffolding (inventory, run, export)
PR-004 Job engine (concurrency, retries)
PR-005 TUI MVP (list, run, logs)
PR-006 Security, keyring & secrets
PR-007 Tests & documentation
PR-008 Packaging & release artifacts

Non-functional requirements:
- Single static binary builds
- JSON machine output for all CLI actions
- Unit tests for core logic
- Linting and formatting enforced in CI
- No `sh -c` for normal ops; use SSH library

Security notes:
- Prefer SSH agent; do not persist plaintext secrets
- Audit logs for all actions
- Interactive confirmation for destructive operations

Acceptance tests:
- inventory import/export
- run command across targets -> structured JSON
- TUI run & live tail

References:
- Bubble Tea ecosystem (examples & components). :contentReference[oaicite:9]{index=9}
- Cobra CLI framework. :contentReference[oaicite:10]{index=10}
- Go SSH library. :contentReference[oaicite:11]{index=11}
```