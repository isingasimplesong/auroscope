# Audit provider contracts — issue #68

## Authority and delivery state

ADR-0066 was accepted by Mathieu's exact `GO DECISION`, comment 3051 of #66.
His comment 3065, `PR de documentation fusionnée, passe à l'implémentation`,
authorizes the execution formalized in #68. This candidate uses one internal
package and a fixed provider switch, not a backend framework or plugin system.

The source is prepared for an immutable pin. Packaging still requires the wrapper
source commit/push, then the real archive checksum, new package identity, non-root
Arch `.SRCINFO`, freshness, upgrade and installed supported-Paru gates. No package
release, deployment, human merge or live-provider qualification is claimed here.

## Shared configuration

Read the root README for exact examples. The only user file is
`${XDG_CONFIG_HOME:-$HOME/.config}/auroscope/config.json`.
It accepts `provider`, `model`, `base_url`, and `api_key_env`. Missing file/provider
keeps Codex and its native model. Existing files are never rewritten. Reads occur
at each AUR audit attempt, not on official-only paths. Unknown fields, duplicate
keys, null fields, invalid UTF-8, trailing JSON and oversized files are rejected.
Errors never quote user-supplied configuration values.

The explicit API model is necessary to construct a valid request; optional CLI
`model` maps to `--model`. This does not settle #65's requested default model.
Prompt configuration and automatic file creation remain #64, in this same file.
No extra model/prompt file or competing environment configuration was introduced.

## Codex CLI

Default invocation, version admission, read-only sandbox, empty stdin and native
credentials remain as recorded in `codex-cli-contract.md`. The optional model is
passed as one argument. The process-group runner is reused for Claude without
changing default Codex progress text. All providers retain local validation.
Final-message file reading is now bounded before allocation, not after it.

## Claude Code

Official documentation consulted:

- <https://code.claude.com/docs/en/cli-reference>
- <https://code.claude.com/docs/en/headless>
- <https://code.claude.com/docs/en/settings-reference#disableallhooks>

Real executable checked: `2.1.269 (Claude Code)`, native Linux x64 npm distribution.
Its tarball and npm integrity metadata were downloaded independently and matched:

- <https://registry.npmjs.org/@anthropic-ai/claude-code-linux-x64/2.1.269>
- <https://registry.npmjs.org/@anthropic-ai/claude-code-linux-x64/-/claude-code-linux-x64-2.1.269.tgz>

```text
sha512-Ti+9oKbf2p9pJuMMj1Fv+6YzljREpy9cx6SN7NV7Zik/vaaXPxC8u+cDiGF7HnPPJ5HsLbmwKoh3BnE3IhdAeQ==
```

The npm parent package now supplies a launcher plus platform-specific optional
packages, not `cli.js`. Use the native platform archive for this contract test.

Production uses `--print --output-format json`, a fresh private directory,
`--no-session-persistence`, `--safe-mode`, and an explicit session name.
`--tools ''`, MCP denial, strict empty MCP configuration, disabled setting sources,
ordinary hooks and slash commands remove package execution capabilities. The
bounded bundle is stdin, never the user's terminal input. The schema is part of
the system prompt; local code parses `result` from a successful result envelope.
It deliberately does not use the agentic structured-output tool.

`--bare` is not interchangeable: current documentation says it disables native
OAuth/keychain authentication. Safe mode preserves native login while suppressing
customizations. Managed administrator settings can still apply; this is not an OS
sandbox or a guarantee against a compromised CLI/administrator. User settings
such as custom authentication helpers are not loaded in this invocation.

The real production client passed in non-root disposable Arch with external
networking disabled and a deterministic loopback Anthropic transport:

- all production flags accepted by the actual executable;
- stdin bundle reached the server; interactive input was not inherited;
- zero tools advertised in the actual model request;
- one request to the selected fixture model;
- actual Claude result envelope decoded and report validated locally.

An initial probe observed an additional title-generation request despite disabled
session persistence. Passing `--name 'AURoscope audit'` removed it; the regression
requires exactly one request. This is an observed CLI contract, not a fake CLI
substituting for the executable. The model response and credential were fixtures:
there was no real Anthropic inference, OAuth login or live model qualification.
Other CLI versions are not qualified; unsupported flags fail without fallback.

The opt-in regression is `TestProviderClaudeRealCLIContract`, with
`AUROSCOPE_TEST_REAL_CLAUDE` pointing to that verified native executable. Run it in
isolated Arch without external networking or real credentials. It changes HOME
only inside its test process and uses a loopback server with an inert credential.

## HTTP contracts

Authoritative sources consulted:

- <https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create>
- <https://github.com/openai/openai-go/blob/main/chatcompletion.go>
- <https://github.com/anthropics/anthropic-sdk-go/blob/main/message.go>
- <https://platform.claude.com/docs/en/build-with-claude/structured-outputs>

These were current online sources, not version-pinned SDK dependencies. Production
uses Go's standard HTTP client, not either SDK. Endpoint/model compatibility still
requires qualification against the actual service.

OpenAI and compatible providers use `/chat/completions` below the configured root,
non-streaming messages, and strict `response_format.json_schema`. Only one completed
assistant choice with `finish_reason: stop`, no refusal and no tool call is accepted.
A service lacking that structured-output contract fails visibly; no schema downgrade
or alternate endpoint is attempted.

Anthropic uses `/v1/messages`, `anthropic-version: 2023-06-01`, `x-api-key`, a system
prompt, an explicit model, 8192 output tokens and `output_config.format`. Its wire
schema omits unsupported `maxLength`, `maxItems` and `minimum`; the original bounds
remain in the prompt and local validation. This follows Anthropic's documented
schema restrictions, not an error-triggered fallback. Only one text block with
`stop_reason: end_turn` is accepted. Tool, refusal and truncated responses fail.

Keys come only from the referenced environment variable and are sent only in the
HTTP authentication header. API URLs require HTTPS without userinfo/query/fragment;
redirects are refused even on the same host. Error bodies and transport diagnostics
are never printed. A valid report reflecting the selected key is discarded before
printing or SQLite persistence. This is exact-key protection, not a universal DLP
system for arbitrary transformed secrets from a malicious remote service.

Requests have cancellation, a timeout, bounded response reads and low-rate progress.
No tools, automatic request retries, provider fallback or approval bypass exist.
The local JSON validator additionally rejects duplicate keys, invalid UTF-8,
missing required top-level fields, null required fields and oversized reports.
Existing file/path/line and terminal-escaping checks remain active.

## Verification and limits

`go test ./...`, `go vet ./...` and race-enabled provider tests passed on the
checkout. Local TLS servers exercise all three API selections, headers, model,
schema, valid JSON, HTTP errors, redirect refusal, cancellation and invalid output.
A subprocess test exercises the built binary's compatible-API configuration,
approval, failed-audit skip and absence of keys in SQLite. The same test supports
`AUROSCOPE_TEST_BINARY=/usr/bin/auroscope` for the installed Arch release gate.

Fake-Claude end-to-end tests prove explicit selection, approval, retry, skip,
cancellation, no Codex fallback and no final build after a failed audit. They also
support the installed artifact. Existing Codex, Paru, drift and auxiliary-script
regressions remain in the full suite. Tests use an isolated XDG configuration root;
they must not read a developer's real provider configuration.

The worker environment has no Anthropic, OpenAI or OpenRouter API key variables;
Claude is not installed there. No personal credentials were searched or copied.
The native Claude test above uses only a downloaded, checksum-matched executable
inside disposable Arch. API simulations do not qualify those live services.

For a separately credentialed live check, use `TestProviderLiveAudit` with:

```sh
AUROSCOPE_TEST_LIVE_PROVIDER=anthropic \
AUROSCOPE_TEST_LIVE_MODEL=YOUR_ACTUAL_MODEL_ID \
CGO_ENABLED=1 go test ./internal/app -run '^TestProviderLiveAudit$' -count=1 -v
```

Use the matching provider and native authentication/environment key. Compatible
APIs also require `AUROSCOPE_TEST_LIVE_BASE_URL` and `AUROSCOPE_TEST_LIVE_KEY_ENV`.
The key variable contains a variable name, never the key itself. The test sends
only inert fixture metadata through the actual production client. Without explicit
opt-in it skips, rather than pretending qualification. Record the exact CLI,
service, endpoint and model after any live run; a provider name is not that proof.
