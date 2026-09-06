# AURoscope Arch package

Self-hosted AUR-style package recipe for AURoscope. Publication on `aur.archlinux.org` is intentionally deferred.

## Prerequisites

- Arch Linux `x86_64`;
- a package providing `paru` with Paru's post-2.1 interactive-output fix and machine order records (currently `paru-git`);
- a supported Codex CLI executable available as `codex` on `PATH`;
- Codex authenticated for the user who runs AURoscope.

AURoscope's currently tested Paru surface is commit `9ac3578807a87858651e81a02586ceb947686e7c`. Stable Paru 2.1.0 is not compatible because it mixes the interactive human menu and selected targets on stdout. The package depends on the virtual `paru` capability so an already installed compatible provider such as `paru-git` satisfies `makepkg`, and conflicts with the known-incompatible stable package version `paru<=2.1.0` so it cannot install into the silent-search failure state. Pacman cannot fetch an absent AUR provider or replace stable `paru` with `paru-git` while installing the already-built AURoscope archive. The recipe therefore stops in `prepare()` with the remediation below when that stable package is installed, before compiling AURoscope. AURoscope also validates the required selection and order behavior when those paths run and fails closed on incompatible output; the package name and `paru --version` string alone do not prove compatibility.

AURoscope admits strict stable Codex CLI versions at least `0.150.1`, with no upper ceiling, including future major versions. Admission is not qualification: the real 0.153.4 audit and direct sandbox checks are documented with their limits in the repository's Codex dependency note; future versions are not automatically tested. Codex is deliberately **not** a Pacman dependency: an installation from npm, an Arch package, or another method works equally as long as `codex --version` reports an admitted version and the user is authenticated.

Examples:

```console
npm install -g @openai/codex
# or, if preferred:
paru -S openai-codex-bin
```

## Install

Bootstrap the package without asking AURoscope to audit itself:

```console
paru -S paru-git
git clone https://git.2027a.net/2027a/auroscope.git
cd auroscope/packaging/aur
makepkg -si
```

If a previous `makepkg -si` offered to remove stable `paru` and then reported that AURoscope's `paru` dependency could not be satisfied, no package was installed. Run `paru -S paru-git` as a separate transaction, then rerun `makepkg -si`.

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

The test builds and installs the package inside disposable Arch, validates its dependency metadata and installed files, then exercises the installed binary's native Paru passthrough, guard failure path, and concise review layout with fake Codex 0.153.4. This reproduces the old package's rejection before validating the new snapshot's admission. The repository root separately carries the full pinned-Paru disposable integration gate and the opt-in real Codex audit test.

## Upstream snapshot

Candidate package `0.1.0.r6.gf97d6ab-1` pins immutable AURoscope commit
`f97d6ab75fbcacf60f17cd02649df6c145795e25`.
It combines native selection colors from #34 with the merged #48 Codex minimum
and retains the earlier #35/#36 fixes. Its archive SHA-256 is:
`d900860363298caa751850edcddef30782a74b152e9d63fa6563a624a5d165f5`

The package build/install gate passed. The full pinned-Paru gate also installs this
exact package and exercises `/usr/bin/auroscope`, including its real selection
color matrix, build/install, edit/re-audit, skip and official-only paths. The drift
scenario returned failure, but its historical assertion is too broad to establish
why it failed; strengthening that evidence is tracked in #50.

PR #49 requires human merge. Preserve the pinned source commit in merge history
rather than squashing it away. Update `_commit`, `pkgver`, and `sha256sums` together
when advancing the executable snapshot. The packaged README comes from the
immutable source and retains its historical candidate status; this recipe's
metadata and current gate evidence describe the newer revision.

Upstream has not yet declared a software license. `LicenseRef-Unspecified` records that fact; it must be replaced when upstream adopts a license.
