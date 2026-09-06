# Dependency note — Codex CLI audit contract

## Context

AURoscope v1 uses Codex CLI as its mandatory audit backend and runs it outside the hostile recipe worktree.

## Verified baselines

Version evidence below is deliberately separate from admission policy. [ADR-0046](../decisions/0046-codex-minimum-version-without-ceiling.md) accepts stable `codex-cli MAJOR.MINOR.PATCH` versions numerically at least `0.150.1`, without a ceiling, including future majors. Older versions, prereleases, and malformed banners are rejected. Execution issue #45 implements this admission policy in the candidate source, not in the still-pinned package. Neither `0.153.4` nor future versions are qualified by admission; real 0.153.4 qualification and the installed-package gate remain incomplete. Keep a pinned contract-test matrix and preserve all invocation, process, and validation protections below. Valid JSON alone does not establish unchanged sandbox/execution semantics.

- Codex CLI: `codex-cli 0.150.1` and `codex-cli 0.151.0`.
- `0.150.1` was verified on 2026-08-31 with the installed executable and a live isolated audit.
- `0.151.0` was verified on 2026-08-31 from the checksum-matched upstream `x86_64-unknown-linux-musl` release artifact inside disposable Arch; its `exec --help` retains every invocation flag used by AURoscope.

## Verified contract

The supported invocation is:

```text
codex exec --json --ephemeral --sandbox read-only --skip-git-repo-check --output-schema <private-schema-path> --output-last-message <private-report-path> <prompt>
```

- `--json` writes JSONL lifecycle events to stdout; stdout is not the audit response.
- `--output-last-message` writes the final assistant message as raw text. AURoscope reads and strictly validates that file as its bounded audit JSON.
- A private non-Git working directory is intentionally used to contain only `bundle.json`; `--skip-git-repo-check` is therefore required. Without it, Codex exits nonzero with `Not inside a trusted directory`.
- `--sandbox read-only` permits reading the bounded bundle while preventing model-driven workspace writes. `workspace-read` is not a valid sandbox value in either supported version.
- `--ephemeral` avoids retaining a Codex session for the package audit.
- `--output-schema` constrains the final response before local validation. Codex/OpenAI structured output requires every object property to appear in `required`; optional finding fields therefore use nullable types and local decoding maps `null` to their zero values.
- `codex exec --help` documents that non-TTY stdin is read and appended even when the prompt is already positional. With an explicit empty stdin, 0.150.1 prints `Reading additional input from stdin...` and immediately sees EOF. That line is not evidence that Codex is waiting for user input. AURoscope must nevertheless provide an isolated empty stdin rather than its interactive Paru/review stream.

The live contract smoke returned JSONL events on stdout and wrote this schema-valid final message to the requested file:

```json
{"summary":"contract smoke","risk":"low","findings":[],"uncertainty":"","inspect":["PKGBUILD"]}
```

## Project usage

AURoscope checks the strict stable banner and numeric minimum before each audit, bounds JSONL/stderr diagnostics and the final-message file, rejects unknown fields and trailing data, and validates every file/line reference against the recipe bundle. It runs Codex in a separate process group with empty stdin, a five-minute timeout, visible 15-second progress, signal forwarding, descendant cleanup, and a fresh private directory for every retry. Any command, version, transport, or validation failure enters the explicit retry/skip/cancel path; it never bypasses the audit silently.

## Verification

- `codex --version` → `codex-cli 0.150.1` for the live isolated audit.
- Checksum-matched Codex `0.151.0` release in disposable Arch → exact version plus all required `exec` flags present.
- Live isolated `0.150.1` invocation with the flags above → exit `0`; JSONL stdout and raw final-message file observed separately.
- Fake-Codex tests reproduce the same argv/output-file contract deterministically for both accepted exact versions.
- The issue #45 regression matrix admits the inclusive minimum, 0.153.4, future minor/major versions, and decimal components beyond machine integer range; it rejects old, prerelease, suffixed, noncanonical, and multiline banners. LF and CRLF line terminators are accepted; other surrounding whitespace is not. Numeric comparison uses canonical decimal length then value, without integer overflow.
- Live `brave-bin` first audit at commit `7b85edf5694c59dd523fabfdec2cedbd525b5e37` with npm-installed Codex 0.150.1 → visible progress at 15-second intervals and a locally schema-valid report after about 45 seconds; Codex consumed no interactive decision input.

## Issue #45 qualification attempt (2026-09-06)

Official release: <https://github.com/openai/codex/releases/tag/rust-v0.153.4>.
The SHA-256 hashes below matched the release API asset digests before execution:

- `codex-x86_64-unknown-linux-musl.tar.gz`: `f479424eca092484dc40d87ae28c44f4cc40234a60045d6131e493800d814a30`.
- `codex-package-x86_64-unknown-linux-musl.tar.gz`: `a822187e1a2420c61c5926721bfbd878701ed95547c9bb0d4de4498a16ba1821`.

The standalone executable reports `codex-cli 0.153.4`; its `exec --help` retains all production flags and the documented stdin append behavior. Its audit attempt also reported a missing sibling `codex-code-mode-host`. The complete official package includes that helper (plus runtime resources); retrying with its `bin/codex` removed that helper error. Do not install only the standalone binary for this contract smoke.

The full-package live test reached the production invocation with read-only sandbox, empty stdin, private bundle/schema/report paths, JSONL diagnostics, and progress after 15 seconds. It failed after 16.86 seconds with CLI exit 1 / HTTP 401 (missing authentication). Profile-isolated `codex login status` also reported `Not logged in`. No other profile's credentials were borrowed, no report was validated, and no successful audit or sandbox behavior is claimed. This is an authentication blocker, not qualification of 0.153.4.

The opt-in test is reproducible with an authenticated environment and a verified full release package:

```console
AUROSCOPE_TEST_REAL_CODEX=/absolute/path/to/codex-package/bin/codex CGO_ENABLED=1 go test ./internal/app -run '^TestCodexReal01534AuditContract$' -count=1 -v
```

It sends only inert fixture metadata, calls the real production client, and requires a locally validated report. It executes no recipe and makes no Paru/Pacman transaction. Without the environment variable, the live test explicitly skips rather than pretending qualification. Successful execution is still not a general sandbox guarantee; preserve the separate process/stdin/timeout and local-validation regression suite.

Local deterministic Go tests and vet pass; the documented fake-Paru disposable-Arch E2E also passes. The installed-package and supported-real-Paru gates were not run: both execute PKGBUILDs, explicitly forbidden in this task. The package source identity is intentionally unchanged pending authentication and permission for the required disposable package gates. This candidate is not a delivered fix for the installed package.
