# Configuration


Use `${XDG_CONFIG_HOME:-$HOME/.config}/auroscope/config.json`. With no file,
AURoscope keeps its original Codex CLI behavior, including Codex's default model.
The provider extension from #68 is included in `main` through merged PR #71.

The file is read only when an AUR audit is required, and reread on explicit retry.
Official operations do not depend on it. AURoscope never overwrites this file.
Automatic creation and editable prompt defaults remain separate work in #64;
the default-model request remains in #65. Both must use this same file.

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

## Other settings

- `EDITOR`: executable for recipe editing, with no embedded arguments (for example `nvim`).
- `AUROSCOPE_PARU` and `AUROSCOPE_CODEX`: executable paths, defaulting to `paru` and `codex` on `PATH`.
- `AUROSCOPE_STATE`: SQLite path, defaulting to `$HOME/.local/state/auroscope/state.sqlite3`.
  The current implementation does not read `XDG_STATE_HOME`.
- `AUROSCOPE_CLONE_DIR`: recipe work directory, defaulting to `auroscope-clones`
  in the system temporary directory.

Each audit has a fixed five-minute timeout. Paru retains its native configuration;
AURoscope adds private settings for recipe worktrees and the final identity guard.
