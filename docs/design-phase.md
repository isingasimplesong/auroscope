# AURoscope design-phase mandate

## Purpose

The next project phase is design, not production implementation. Its job is to turn the accepted product specification into grounded technical proposals, submit consequential choices to Mathieu, record accepted decisions, and only then produce an implementation plan.

## Fixed constraints

- Implementation language: Go.
- Deliverable: a small `auroscope` executable, initially targeting Arch Linux on `linux/amd64`.
- Dependencies: prefer the Go standard library; add only ordinary, maintained dependencies that avoid materially worse reinvention. Every direct dependency requires a concrete justification.
- Distribution: a self-hosted AUR-style PKGBUILD repository first; publication on `aur.archlinux.org` is deferred.
- Go should be a build dependency, not a runtime dependency, where feasible.
- SQLite is the authoritative durable state store.
- Human-readable status is rendered from SQLite; exported Markdown/JSON is not read back as state.
- Temporary recipe clones and audit inputs must not accumulate. Use private directories under `$TMPDIR` and/or bounded reconstructible cache, clean on success/error/interruption, and provide stale-work cleanup for crash residue.
- Interactive decisions always belong to the user. Deterministic findings and LLM assessments remain separate advisory evidence.
- Do not execute or source untrusted PKGBUILDs during collection or review.

## Standard storage roots

```text
${XDG_CONFIG_HOME:-$HOME/.config}/auroscope/
${XDG_STATE_HOME:-$HOME/.local/state}/auroscope/
${XDG_CACHE_HOME:-$HOME/.cache}/auroscope/
${TMPDIR:-/tmp}/auroscope-*/
```

The design must assign each concrete artifact to the correct root. `/tmp` is not assumed to be RAM; the hard requirement is privacy, bounded lifetime, and no silent accumulation.

## Required investigations and proposals

### 1. Exact Paru/Pacman contract

Inspect the exact supported Paru/Pacman versions, local/versioned manuals, source, and tests. Propose:

- the argument classification table: transparent pass-through versus intercepted flows;
- capture of Paru-native interactive selections without recreating its TUI;
- authoritative origin and dependency resolution;
- complete official-repository upgrades while deferring selected AUR work;
- handling of explicit installs, upgrades, removals, queries, caches, already-built packages, PKGBUILD repositories, cancellation, signals, and exit codes;
- minimum supported versions and compatibility boundaries.

### 2. Go architecture and dependencies

Propose modules and process boundaries, CLI parsing, configuration loading, subprocess/PTY handling, cancellation, structured errors, and build/release strategy. Compare SQLite driver options, especially pure-Go versus CGO consequences. Do not add a framework merely to create an architecture diagram.

### 3. SQLite state model

Propose tables, keys, constraints, migrations, transactions, concurrency/locking, crash recovery, lifecycle transitions, retention, and purge. Distinguish recipe identity, inspection, findings, assessment, human decision, approval, build, and installation.

### 4. Approval and TOCTOU protocol

Define the exact commit/hash identity, approval scope and expiry, wrapper-to-`PreBuildCommand` handoff, anti-replay behavior, invalidation, cache handling, modified recipes, and cleanup after every terminal state.

### 5. Deterministic scanner and LLM contracts

Propose versioned schemas, initial rule catalogue, evidence semantics, context selection/truncation, prompt-injection boundaries, model/provider configuration, privacy, timeout/failure behavior, and logging. The LLM must never rewrite or suppress deterministic findings.

### 6. Configuration and XDG layout

Propose the config format, defaults, exact paths, permissions, report generation, editor/pager behavior, bounded cache policy, and temporary-work cleanup.

### 7. Threat model

Cover hostile Git repositories and recipes, symlinks/path traversal, prompt injection, audit/build substitution, PATH/editor/config manipulation, sudo and privilege boundaries, process concurrency, Pacman locking, secrets, reports, and crash residue.

### 8. Test strategy

Propose unit, contract, PTY, SQLite/migration, scanner, LLM, TOCTOU, failure, and disposable-Arch end-to-end tests. Keep the plan proportional, but require real integration proof for security boundaries.

## Process and deliverables

1. Read `README.md`, `docs/specification.md`, this document, and `AGENTS.md`.
2. Revalidate third-party behavior from exact source/docs/tests; do not rely on model memory.
3. Consult `2027a/paru-llm-audit` only for targeted lessons, fixtures, and pitfalls—not as architecture to port.
4. Run narrow disposable spikes where documentation cannot establish behavior.
5. Write proposals with options, trade-offs, recommendation, unresolved risks, and proof obtained.
6. Submit consequential choices to Mathieu before declaring them accepted.
7. Record accepted choices as ADRs under `docs/decisions/`.
8. Produce the detailed implementation plan only after the design decisions are accepted.
9. Do not start production implementation during this phase unless Mathieu explicitly advances the phase.

## Decision issue protocol

Consequential design choices are discussed in Forgejo issues whose first non-empty line is `MODE: DECISION`.

- Ordinary comments continue a bounded issue discussion; they do not authorize repository changes, ADR acceptance, implementation, or closure.
- Mathieu accepts the latest concrete proposal by posting a standalone comment exactly equal to `GO DECISION`.
- The watcher validates Mathieu's pinned Forgejo identity and requires the GO to be the latest external comment, then transfers the issue from `Agent/Human` to `Agent/Hermes` for ADR/docs finalization.
- Finalization records and verifies the decision, but cannot write production code or the implementation plan.
- Executable follow-up work uses a separate issue whose first non-empty line is `MODE: EXECUTION`.
- Missing or contradictory routing fails closed under `Agent/Needs Review`.

Expected design artifacts may include:

```text
docs/development-frame.md
docs/architecture/paru-contract.md
docs/architecture/go-and-dependencies.md
docs/architecture/sqlite-state-model.md
docs/architecture/approval-protocol.md
docs/architecture/scanner-llm-contracts.md
docs/architecture/threat-model.md
docs/architecture/test-strategy.md
docs/decisions/NNNN-*.md
docs/implementation/initial-plan.md   # only after design approval
```

The list is guidance, not permission to create empty document-shaped bureaucracy. Combine documents when that improves clarity.
