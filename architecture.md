I recommend building the app in **Go** using the **Bubble Tea** ecosystem (Bubble Tea for app architecture, Bubbles for components, Lip Gloss for styling) together with **Cobra** for command parsing and `golang.org/x/crypto/ssh` for SSH interactions. Bubble Tea is a mature, Elm-style TUI framework for Go; it compiles to a single machine binary, supports rich styling components, and is widely used for modern TUIs.

---

# 1 — Top-level decision (short)

* **Language:** Go (compiled, simple concurrency model, strong standard toolchain and cross-compile support). ([blackthorn-vision.com][2])
* **TUI stack:** Bubble Tea + Bubbles + Lip Gloss (modern TUI look & UX). ([GitHub][1])
* **CLI scaffolding & non-TUI commands:** Cobra (standard de-facto CLI framework in Go). ([GitHub][3])
* **SSH:** `golang.org/x/crypto/ssh` (do not shell out by default — use a library or a small wrapper to support agent/key parsing). ([Go Packages][4])

Rationale: performance, single static binary delivery, safer memory model vs C, structured code (types/structs), excellent tooling for formatting, linting and testing. Bubble Tea gives you polished terminal UI components (the look-and-feel you requested) and a maintainable Elm-style message/update architecture. ([GitHub][1])

---

# 2 — High-level architecture

Components (modules/packages)

* `cmd/` — Cobra entrypoints (subcommands): `vps-tools`, `vps-tools tui`, `vps-tools run`, `vps-tools export`, `vps-tools audit`.
* `internal/app` — application core (domain logic orchestrating operations).
* `internal/tui` — Bubble Tea app: Views, models, components (table, logs, forms, progress).
* `internal/ssh` — SSH client abstraction (connect, run, stream, scp), wraps `golang.org/x/crypto/ssh` and optionally supports shelling out to `ssh` as a fallback.
* `internal/server` — `Server` domain model + manager: struct Server {ID, Hostname, IP, Port, User, Auth (agent/key), Tags, Status, LastSeen, LastJobResult}.
* `internal/store` — local state: configuration, credential metadata, cache (BoltDB or SQLite). (Choose BoltDB if simplicity and embedded key-value fits; SQLite if you prefer relational queries.)
* `internal/jobs` — job scheduling/execution engine (concurrent goroutines, queue, retry, timeouts).
* `internal/audit` — append-only audit log + activity stream for commands and results.
* `pkg/ui/components` — reusable Bubble Tea components (tables, forms, spinners, selectable lists) — lever Bubbles.
* `scripts/migrations` — small scripts for migrating existing bash outputs/config into new data model (e.g., parse inventory files).
* `ci/` — GitHub Actions pipeline: build, test, cross-compile, static analysis, release artifacts.

Data flows

1. CLI receives command (Cobra).
2. App parses config (YAML/TOML) and inventory (existing scripts → migrate to structured config).
3. Commands call `internal/app` which uses `internal/server` + `internal/ssh` + `internal/jobs`.
4. TUI binds to `internal/app` via message bus and displays states (Bubble Tea).
5. Results update `internal/store` and `internal/audit`.

Security, sandboxing & process model

* No arbitrary `sh -c` execution unless explicitly required; prefer native SSH operations via library to avoid shell injection.
* Explicit user consent & config for private key usage; prefer SSH agent or encrypted key store.
* Least privilege — commands that change server state must require confirmation (interactive) or `--yes` flag.
* Optionally run remote commands under a dedicated, minimal account (document this in ops).
* Audit trail for every action (who ran it, when, on which server, command + stdout/stderr hashes).

Deployment and packaging

* Single static binary per platform (Linux, macOS, Windows). Use Go’s cross-compile or GitHub Actions matrix.
* Provide both: CLI (non-interactive, scriptable) and TUI (interactive).
* Releases: GitHub Releases + checksums + optionally Homebrew tap / APT repo instructions.

Observability & logging

* Structured logs (JSON) and human logs (for TUI). Log rotation for local logs.
* Optional telemetry (opt-in) for anonymous crash reports; no sensitive data.

---

# 3 — Data model (core)

Server (Go struct, example)

```go
type Server struct {
    ID         string            // uuid
    Name       string
    Host       string            // IP/hostname
    Port       int
    User       string
    AuthMethod AuthConfig        // agent / privateKey / password (avoid password storage)
    Tags       []string
    Status     ServerStatus
    LastSeen   time.Time
    Meta       map[string]string
}
```

Job

```go
type Job struct {
    ID        string
    ServerID  string
    Command   string
    StartedAt time.Time
    FinishedAt *time.Time
    ExitCode  int
    Stdout    string
    Stderr    string
    Status    JobStatus
}
```

Config

* `~/.config/vps-tools/config.yaml` — inventory, default SSH options, concurrency limits, retention policy.
* Secrets: never store raw private keys without user explicit opt-in; support encrypted store with passphrase or integration with OS keyring.

---

# 4 — Migration strategy from Bash scripts

Goal: translate existing scripts to small pure functions and connect to new domain model rather than line-by-line rewrite.

Phased approach (each phase is a deliverable for the AI coding tool):

Phase A — Discovery & adapters

* Inventory: scan repo for bash scripts, parse usage (input args, env vars, common patterns).
* Create an **adapter** library that can run the existing script and capture structured output (temporary), to avoid disrupting operations during migration.

Phase B — Implement core platform

* Implement `Server` and `SSH` abstraction.
* Implement simple CLI commands that replicate script behaviour (e.g., `vps-tools status`, `vps-tools update`, `vps-tools backup`) that call SSH wrapper functions.

Phase C — Replace scripts with typed modules

* Reimplement script logic as Go functions (idempotent, testable).
* Create unit tests matching previous outputs (use adapter outputs as golden files).

Phase D — TUI & polishing

* Create Bubble Tea TUI for common admin tasks: inventory browser, job runner, live logs, interactive shell to server.
* Style with Lip Gloss and use Bubbles for components.

Phase E — Hardening & release

* Harden SSH config parsing, key handling, secrets encryption.
* CI, cross-compile, packaging, changelog and migration docs.

