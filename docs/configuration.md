# Configuration


Use `${XDG_CONFIG_HOME:-$HOME/.config}/auroscope/config.json`. With no file,
AURoscope uses Codex CLI and explicitly requests `gpt-5.6-luna`.
The provider extension from #68 is included in `main` through merged PR #71.

The file is read only when an AUR audit is required, and reread on explicit retry.
Official operations do not depend on it. AURoscope never overwrites this file.
On the first AUR audit, a missing file is created with the complete built-in
audit prompt in `prompt`. For Codex, an omitted `model` selects `gpt-5.6-luna`; an
explicit nonempty `model` is passed unchanged as one `--model` argument. For
example, `{"model":"your-exact-codex-model-id"}` selects another model. Empty
or null models are rejected. Claude Code retains its native default when the
model is omitted; API providers require an explicit model. AURoscope does not
translate model aliases or fall back. Access to `gpt-5.6-luna` and real inference
quality have not been qualified; use a model available to your account.
An existing explicit `"model":"luna"` is not rewritten: change it to
`"model":"gpt-5.6-luna"` or omit `model` to use the corrected default.

For Codex, optional `thinking` sets its native `model_reasoning_effort`:

```json
{"provider":"codex","model":"gpt-5.6-luna","thinking":"medium"}
```

Omit `thinking` to leave Codex's own default/configuration unchanged. AURoscope
passes the string literally, without trimming or checking a list of levels;
you are responsible for choosing a value accepted by your Codex and model.
Empty or unknown strings are also sent to Codex, whose errors retain the normal
retry/skip/cancel path. Null and non-string values are rejected locally.
This field is Codex-only; other providers reject it rather than silently ignoring it.
Existing configuration is never rewritten, and explicit retry rereads this field.

Choose exactly one provider. Minimal CLI configurations are:

```json
{"provider": "codex"}
```

```json
{"provider": "claude-code"}
```

CLI authentication stays with the CLI; no API key is stored by AURoscope. Claude
Code uses the `claude` executable on `PATH`, with tools, MCP, user/project settings,
skills and ordinary hooks disabled for the audit. Native login credentials remain
available. Model selection is optional for CLI providers via `model`.

For API providers, specify an actual model identifier available to your account:

```json
{
  "provider": "anthropic",
  "model": "YOUR_MODEL_ID",
  "api_key_env": "ANTHROPIC_API_KEY"
}
```

For OpenAI, use `"provider": "openai"` and `"api_key_env": "OPENAI_API_KEY"`.
These providers use their official HTTPS API roots. For another service:

```json
{
  "provider": "openai-compatible",
  "base_url": "https://openrouter.ai/api/v1",
  "model": "YOUR_PROVIDER/MODEL_ID",
  "api_key_env": "OPENROUTER_API_KEY"
}
```

`base_url` is the API root, not the full `/chat/completions` endpoint. HTTPS is
required; credentials in URLs, query strings, fragments and redirects are refused.
The referenced environment variable must hold the key when AURoscope starts.
Never put a key value in this file. Unknown fields and invalid configuration fail
the audit rather than selecting a different provider. The file is limited to 64 KiB.

API selection sends the same bounded recipe bundle to that service. OpenAI and
compatible services must support Chat Completions with `response_format.json_schema`;
Anthropic must support Messages with `output_config.format`. Refusal, truncation,
HTTP error or invalid local report validation offers retry, skip or cancel only.
There is no automatic provider fallback, schema-mode downgrade or audit bypass.

Compatibility is not guaranteed by a provider name. Claude Code 2.1.269 was checked
with its real executable and simulated transport in isolated Arch. API contracts
have deterministic local HTTPS tests; no new live service/model is qualified yet.
See the [provider contract note](dependency-notes/audit-providers.md) for exact
evidence, limitations and opt-in live qualification commands.

## Editable audit prompt

Edit the `prompt` JSON string in the same configuration file as provider and
model settings. Use `\n` for line breaks. The string is passed to the selected
provider; it does not replace the bounded recipe bundle or local report checks.

Existing files are never rewritten. If an existing file omits `prompt`, the
built-in default is used without modifying the file. An empty or null prompt,
invalid JSON, or an invalid configuration fails the audit with retry/skip/cancel,
never approval. The file is reread on explicit retry. Save a copy of a custom
prompt before deleting the file to regenerate the default on the next AUR audit.

Prompt customization does not disable read-only CLI protections, strict local
report validation, human approval, or the final recipe-identity guard. Package
upgrades do not own or replace user configuration.

## Other settings

- `EDITOR`: executable for recipe editing, with no embedded arguments (for example `nvim`).
- `AUROSCOPE_PARU` and `AUROSCOPE_CODEX`: executable paths, defaulting to `paru` and `codex` on `PATH`.
- `AUROSCOPE_STATE`: SQLite path, defaulting to `$HOME/.local/state/auroscope/state.sqlite3`.
  The current implementation does not read `XDG_STATE_HOME`.
- `AUROSCOPE_CLONE_DIR`: recipe work directory, defaulting to `auroscope-clones`
  in the system temporary directory.

Each audit has a fixed five-minute timeout. Paru retains its native configuration;
AURoscope adds private settings for recipe worktrees and the final identity guard.
