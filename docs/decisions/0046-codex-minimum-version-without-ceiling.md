# ADR-0046: Codex minimum version without a ceiling

## Status

Accepted on 2026-09-06 in [decision issue #46](https://git.2027a.net/2027a/auroscope/issues/46), by Mathieu's exact standalone [GO DECISION, comment 1845](https://git.2027a.net/2027a/auroscope/issues/46#issuecomment-1845). The latest concrete proposal is [comment 1844](https://git.2027a.net/2027a/auroscope/issues/46#issuecomment-1844), retaining the contract detailed in [comment 1830](https://git.2027a.net/2027a/auroscope/issues/46#issuecomment-1830).

This records design acceptance only. Implementation, real qualification of Codex 0.153.4, and a new packaged release require a separate `MODE: EXECUTION` issue. At decision acceptance, the implementation admitted only 0.150.1 and 0.151.0. The separate execution candidate in issue #45 now implements admission; real qualification and the updated installed-package gate remain pending, and the package still pins the earlier implementation. ADR-0015's mandatory Codex audit and human-authority boundaries remain unchanged.

## Context and evidence

The desktop report in [issue #45](https://git.2027a.net/2027a/auroscope/issues/45) reached the Codex version gate after the initial Paru problem was reported resolved. At repository commit `f5310e5`, `internal/app/codex.go` admits only the exact banners `codex-cli 0.150.1` and `codex-cli 0.151.0`; this rejects 0.153.4 solely because its version is unknown.

The issue records a temporary fake-version test reproducing that refusal. This is evidence of the admission rule, not evidence of real 0.153.4 compatibility. Existing version-specific evidence remains in the [Codex dependency note](../dependency-notes/codex-cli-contract.md): a live isolated 0.150.1 audit and checksum-matched 0.151.0 artifact/help checks. No future version is qualified by this decision.

## Options and trade-offs

1. Keep an exact list and qualify/add 0.153.4: conservative, but each new version may need another AURoscope release merely to pass the version gate.
2. Use an inclusive minimum of 0.150.1 without a ceiling (recommended and accepted): avoids that administrative refusal, while accepting maintenance risk from unqualified CLI versions.
3. Impose a major-version ceiling: would still permit breaking 0.x minor changes and recreate a version-only refusal at 1.x; not selected.

## Decision

- Admit stable versions with the strict banner `codex-cli MAJOR.MINOR.PATCH` and a numerically compared version triplet greater than or equal to `0.150.1`.
- Impose no upper bound, including on future stable major versions.
- Reject older versions, prereleases, and malformed or unexpected banners.
- Distinguish **admitted by policy** from **qualified by tests** in documentation and verification evidence. An admitted version is eligible for an audit attempt, not guaranteed compatible or safe.
- Preserve the current Codex invocation and protections: `--sandbox read-only`, a private working directory outside the recipe, isolated empty stdin, bounded diagnostics/output, timeout, signal handling, and process cleanup.
- Preserve strict local report validation, including schema, sizes, paths, and line references. A CLI error or invalid report remains retry/skip/cancel; it is never a successful audit, automatic approval, or an audit bypass.
- Keep a pinned version matrix for contract tests. This policy does not change Paru compatibility or the final recipe-identity guard.

## Required verification before delivery

The separate execution lane must cover the inclusive minimum and its lower boundary, 0.153.4, future stable minor/major versions, numeric rather than lexical comparison, and rejected prerelease/malformed banners. It must also retain CLI/JSON failure and no-bypass tests, perform real qualification of Codex 0.153.4 with the actual invocation and protections, document exactly which versions and behaviors were tested, and update the pinned package with a new identity and installed-package gate.

These are acceptance obligations, not an implementation plan or a claim that the checks have already passed.

## Consequences and unresolved risks

A routine Codex update no longer requires an AURoscope release solely to add a number to a list. In exchange, AURoscope may attempt an unqualified CLI. A valid JSON report cannot by itself prove unchanged sandbox or execution semantics. Future CLI changes may require maintenance and new contract tests; the version minimum is not a security or compatibility guarantee. Codex failure must remain visible and must never silently authorize a package.
