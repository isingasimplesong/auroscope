# Dependency note — Codex CLI audit contract

## Context

AURoscope v1 uses Codex CLI as its mandatory audit backend and runs it outside the hostile recipe worktree.

## Verified baselines

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

AURoscope checks the exact version string before each audit, bounds JSONL/stderr diagnostics and the final-message file, rejects unknown fields and trailing data, and validates every file/line reference against the recipe bundle. It runs Codex in a separate process group with empty stdin, a five-minute timeout, visible 15-second progress, signal forwarding, descendant cleanup, and a fresh private directory for every retry. Any command, version, transport, or validation failure enters the explicit retry/skip/cancel path; it never bypasses the audit silently.

## Verification

- `codex --version` → `codex-cli 0.150.1` for the live isolated audit.
- Checksum-matched Codex `0.151.0` release in disposable Arch → exact version plus all required `exec` flags present.
- Live isolated `0.150.1` invocation with the flags above → exit `0`; JSONL stdout and raw final-message file observed separately.
- Fake-Codex tests reproduce the same argv/output-file contract deterministically for both accepted exact versions.
- Live `brave-bin` first audit at commit `7b85edf5694c59dd523fabfdec2cedbd525b5e37` with npm-installed Codex 0.150.1 → visible progress at 15-second intervals and a locally schema-valid report after about 45 seconds; Codex consumed no interactive decision input.
