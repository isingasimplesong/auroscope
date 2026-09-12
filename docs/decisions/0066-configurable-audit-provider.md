# ADR-0066: Explicit audit provider selection

## Status

Accepted on 2026-09-12 in [decision issue #66][issue], through Mathieu's exact
standalone [GO DECISION, comment 3051][acceptance], accepting the concrete
[proposal in comment 3046][proposal].

This is design acceptance only. Production still uses Codex CLI exclusively.
Implementation and Arch delivery require a separate `MODE: EXECUTION` issue;
this decision does not authorize an implementation plan or production changes.

This amends ADR-0015 point 4 only for provider selection. ADR-0046 continues to
control Codex version admission and its distinction from qualification.

## Context

Mathieu wants to retain Codex by default while choosing another audit provider
in the configuration file. The previous Codex-only and no-HTTP restriction
prevented that choice. The LLM remains mandatory, not optional enrichment.

## Decision

- Keep Codex CLI as the default, unchanged for an unconfigured installation.
- Allow explicit selection of one provider in the configuration file: Codex CLI,
  Claude Code, Anthropic API, OpenAI API, or an OpenAI-compatible API with a
  configurable URL, including services such as OpenRouter or Zen.
- Reference API keys through environment variables. Do not put their values in
  recipes, reports, or logs. CLI providers retain their native authentication.
- Keep the exact recipe audit scope, strict local report validation, human
  decision, and final recipe-identity check. Never execute package-supplied
  content during audit.
- If the selected provider fails or produces invalid output, offer retry, skip
  the package, or cancel. Never switch providers automatically or bypass audit.
- Leave Paru/Pacman responsibilities and audit-free official packages unchanged.

Configurable model and prompt remain the subjects of issues #65 and #64.
Their integration must share the same configuration file, not introduce
competing configurations. This decision does not select a format or keys.

## Consequences and delivery boundary

Users can choose their audit service without changing the approval workflow.
Remote API use sends the audit bundle to the explicitly selected service;
provider choice does not enlarge the recipe data collected for that bundle.

Compatibility is not established by an OpenAI-compatible label. No new provider,
endpoint, model, or CLI version is qualified by this decision. Execution must
establish the actual invocation and response contracts and preserve the audit
protections; it must not claim a successful audit from provider admission alone.

The separate execution lane must verify the unchanged default, explicit choice,
secret handling, local report validation, and failure without fallback or bypass.
Shipped input changes require the immutable source pin, real archive checksum,
new package identity, non-root Arch `.SRCINFO`, freshness and upgrade/package
gates, and the installed-artifact supported-Paru gate. Source-only checks do not
qualify the package. No installation on Mathieu's machine is authorized here.

The present documentation change ships no code, dependency, test, or root README
change and requires no new binary release. Publication and human review remain
separate from acceptance; the requested provider feature is not yet delivered.

[issue]: https://git.2027a.net/2027a/auroscope/issues/66
[proposal]: https://git.2027a.net/2027a/auroscope/issues/66#issuecomment-3046
[acceptance]: https://git.2027a.net/2027a/auroscope/issues/66#issuecomment-3051
