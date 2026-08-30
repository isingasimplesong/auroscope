# ADR-0001: Go implementation and self-hosted Arch packaging

## Status

Accepted

## Context

AURoscope should replace Paru as the user's normal terminal entry point while remaining easy to install and distribute. It needs robust subprocess, signal, timeout, terminal, structured-state, and SQLite behavior without requiring a language runtime on the target machine.

## Decision

- Implement AURoscope in Go.
- Initially target Arch Linux on `linux/amd64`; additional target binaries may be produced later.
- Prefer the Go standard library and use only minimal, maintained, justified direct dependencies.
- Evaluate the SQLite driver during design, including pure-Go and CGO trade-offs.
- Distribute initially through a self-hosted AUR-style PKGBUILD repository.
- Treat Go as a build dependency rather than a runtime dependency where feasible.
- Keep publication to `aur.archlinux.org` out of the initial scope.

## Consequences

Positive:

- straightforward single-executable installation per target platform;
- no Python or other application runtime required;
- strong standard-library support for process orchestration and concurrency;
- typed contracts for findings, assessments, decisions, and persistent state.

Negative or deferred:

- binaries are target-specific rather than universally portable;
- the SQLite driver choice may affect binary size, CGO requirements, and reproducibility;
- PTY behavior and exact Paru integration still require grounded design spikes;
- dependency minimization must not become bespoke replacements for mature fundamentals.
