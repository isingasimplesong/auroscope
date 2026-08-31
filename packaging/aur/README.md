# AURoscope Arch package

Self-hosted AUR-style package recipe for AURoscope. Publication on `aur.archlinux.org` is intentionally deferred.

## Prerequisites

- Arch Linux `x86_64`;
- Paru 2.1.0 or a compatible package providing `paru`;
- a supported Codex CLI executable available as `codex` on `PATH`;
- Codex authenticated for the user who runs AURoscope.

AURoscope currently accepts the exact Codex CLI versions `0.150.1` and `0.151.0`. Codex is deliberately **not** a Pacman dependency: an installation from npm, an Arch package, or another method works equally as long as `codex --version` reports a supported version.

Examples:

```console
npm install -g @openai/codex
# or, if preferred:
paru -S openai-codex-bin
```

## Install

Bootstrap the package without asking AURoscope to audit itself:

```console
git clone https://git.2027a.net/2027a/auroscope.git
cd auroscope/packaging/aur
makepkg -si
```

Then verify the installed paths:

```console
command -v auroscope
codex --version
paru --version
```

## Use

```console
auroscope                    # official update, then audited AUR updates
auroscope <search terms>     # Paru-native interactive selection
auroscope -S <packages>      # explicit package installation
auroscope --version          # transparent Paru passthrough
```

State is stored in `${XDG_STATE_HOME:-$HOME/.local/state}/auroscope/state.sqlite3`. Codex authentication remains in Codex's own user configuration.

During the first desktop trial, keep invoking `auroscope` explicitly rather than replacing `paru` with an alias.

## Package verification

From `packaging/aur` on a machine with Docker:

```console
scripts/test-package.sh
```

The test builds and installs the package inside disposable Arch, validates its dependency metadata and installed files, then exercises the installed binary's native Paru passthrough and guard failure path. The repository root separately carries the full pinned-Paru disposable integration gate.

## Upstream snapshot

This recipe pins immutable AURoscope commit `6e02c6ddf27ae398ebea3025758b894e526b2eff`, including the compact interactive review and packaging-focused Codex audit contract from issue #31. Update `_commit`, `pkgver`, and `sha256sums` together when advancing the package.

Upstream has not yet declared a software license. `LicenseRef-Unspecified` records that fact; it must be replaced when upstream adopts a license.
