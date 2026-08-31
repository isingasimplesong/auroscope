# Dependency note — Go SQLite driver comparison

## Context

AURoscope uses SQLite as its authoritative store and initially targets Arch Linux on `linux/amd64`. Decision D6 compared an embedded CGO driver with a pure-Go driver before selecting the v1 dependency.

## Versions investigated

- `github.com/mattn/go-sqlite3 v1.14.50`
- `modernc.org/sqlite v1.57.0`
- spike host: Go `1.24.4`
- disposable Arch observation: Go `2:1.27.0-1`

## Verified findings

The same minimal program was used to create a database, enable `PRAGMA foreign_keys`, select WAL mode, and verify a foreign-key constraint.

| Driver | Stripped binary | Modules | Result |
|---|---:|---:|---|
| `mattn/go-sqlite3` | 3,608,224 bytes | 2 | works with CGO; a `CGO_ENABLED=0` build compiles a stub that is unusable at runtime |
| `modernc.org/sqlite` | 6,299,940 bytes | 26 | pure-Go runtime works and cross-compiles; tested version requires Go 1.25+ |

For the single Arch `linux/amd64` v1 target, the measured pure-Go portability benefit does not justify the larger binary and module graph. External SQLite CLI calls are not an equivalent driver: they weaken transaction composition and structured error handling.

## Project usage

[`ADR-0007`](../decisions/0007-mattn-go-sqlite3-cgo.md) accepts `github.com/mattn/go-sqlite3 v1.14.50` with CGO. Pin the module and SQLite compile options. Release verification must exercise the database at runtime so a `CGO_ENABLED=0` stub cannot pass on compilation alone.

Revisit the driver only if cross-target releases become an actual supported requirement.

## Sources consulted

- `github.com/mattn/go-sqlite3` source/documentation at the tested `v1.14.50` module version
- `modernc.org/sqlite` source/documentation at the tested `v1.57.0` module version
- [`docs/archive/pre-llm-first/design-proposals.md`](../archive/pre-llm-first/design-proposals.md), including the recorded disposable spike
- [decision issue #7](https://git.2027a.net/2027a/auroscope/issues/7)

## Required verification in the implementation lane

- runtime database smoke test with the release build;
- foreign-key enforcement and WAL;
- busy timeout and concurrent access;
- migrations and integrity/corruption handling;
- backup and restore;
- recorded driver version and SQLite compile options.
