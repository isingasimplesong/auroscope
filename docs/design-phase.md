# AURoscope design status

## Current phase

The minimal LLM-first architecture was accepted by Mathieu in [decision issue #19](https://git.2027a.net/2027a/auroscope/issues/19) with an exact standalone [`GO DECISION`](https://git.2027a.net/2027a/auroscope/issues/19#issuecomment-1314) on 2026-08-30.

Production implementation was authorized separately by Forgejo issue #24 on branch `work/issue-24-implement-auroscope-v1-from-the-merged-initial-p`. The issue #24 lane implements the accepted minimal v1 and verifies it with deterministic fake-Paru/fake-Codex tests plus the guarded pinned-Paru disposable-Arch E2E.

Before that authorization, the next permitted work was:

1. a narrow disposable-Arch spike proving the Paru worktree/audit/edit/final-build path;
2. a new short implementation plan grounded in that evidence;
3. separate explicit authorization before production implementation.

## Accepted product frame

Decision #66 is accepted through Mathieu's exact standalone
[GO DECISION, comment 3051](https://git.2027a.net/2027a/auroscope/issues/66#issuecomment-3051).
[ADR-0066](decisions/0066-configurable-audit-provider.md)
records Codex by default and explicit choice of Claude Code, Anthropic API,
OpenAI API, or an OpenAI-compatible API. It amends the Codex-only product frame
below, not the mandatory audit or Paru responsibilities. Production remains
Codex-only until a separate `MODE: EXECUTION` issue authorizes implementation
and Arch delivery. This finalization contains documentation only, not an
implementation plan, provider qualification, or permission to deploy.

The later [decision #46](https://git.2027a.net/2027a/auroscope/issues/46), accepted by Mathieu's exact [GO DECISION](https://git.2027a.net/2027a/auroscope/issues/46#issuecomment-1845), is recorded in [ADR-0046](decisions/0046-codex-minimum-version-without-ceiling.md): Codex minimum `0.150.1`, no ceiling, strict stable banners, unchanged execution/validation protections, and explicit separation of admission from qualification. That decision finalization changed documentation only. Issue #45 is the separate `MODE: EXECUTION` lane implementing the admission candidate; its installed-package gate passed, together with the real 0.153.4 audit and bounded sandbox checks documented in the dependency note. The candidate remains under human PR review, not automatically merged or installed on Mathieu's workstation. Paru policy is unchanged.

AURoscope has one purpose: audit exact AUR recipe changes with Codex CLI before Paru builds them, present the result, and collect the user's per-package decision.

- Official packages remain native Paru/Pacman operations and are never audited.
- Search, selection, resolution, build, and installation remain Paru responsibilities.
- Codex CLI is the only v1 LLM backend and is not optional in an AUR audit.
- AURoscope keeps only minimal audit/baseline state and a final recipe-identity guard.
- The local machine and account are trusted.

See [`architecture/minimal-v1.md`](architecture/minimal-v1.md), [`specification.md`](specification.md), and [`ADR-0015`](decisions/0015-minimal-llm-first-wrapper.md).

## Required spike

The new plan must not guess the central Paru integration. In disposable Arch, prove:

1. native search/selection and AUR origin identification;
2. the exact AUR worktree is available before package-supplied code;
3. an edited recipe can be re-audited and the same worktree reaches final build, or `edit` is explicitly deferred;
4. `PreBuildCommand` sees the exact worktree before build;
5. skipped AUR targets can be excluded and Paru performs the final fresh resolution;
6. official repository operations remain transparent and audit-free.

## Decision protocol

Consequential design issues begin with `MODE: DECISION`. Ordinary comments continue discussion only. Mathieu accepts the latest proposal by posting a standalone comment exactly equal to `GO DECISION`.

Finalization may update ADRs and design/specification documents, but does not authorize an implementation plan or production code. Executable follow-up uses a separate issue whose first non-empty line is `MODE: EXECUTION`.

## Historical material

The previous architecture made the LLM optional and introduced a deterministic scanner, broad state model, durable approval protocol, closure comparison, and extensive recovery machinery. It is superseded and archived under [`archive/pre-llm-first/`](archive/pre-llm-first/). Historical documents are evidence only, not instructions.
