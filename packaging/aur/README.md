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

The test checks source freshness, builds the previous and current packages in
Arch, and upgrades with the old archives still present. It validates dependency
metadata and installed files, then exercises native Paru passthrough, guard
failure, and review layout with fake Codex 0.153.4. The repository root carries
the full pinned-Paru integration gate and the opt-in real Codex audit test.

## Upstream snapshot

Candidate package `0.1.0.r9.g3c5f935-1` pins immutable AURoscope commit
`3c5f935e71f88445290522119f4cb2f8b56f9e97`.
It descends from current `main` (`8d9b844c093269286a8e5d2818d77665fecf7e99`),
retains the previous fixes and includes merged PR #61: clone preparation
neither traverses nor chmods unreadable build artifacts below the clone root.
It also makes the disposable gate work from linked Git worktrees by copying
source without host `.git` metadata, with a checkout/worktree regression test.
Its archive SHA-256 is:
`9eff0b666631389abb79f0b38731bab6b4545f6506d86cfcfbfc0f443fe5f3c6`

On 2026-09-12, the disposable Arch package gate built and installed the previous
`0.1.0.r8.gc338567-1` package, left its real archives beside the new recipe, then
ran `makepkg --syncdeps --install --noconfirm` without `--force`. Pacman upgraded
both AURoscope and its debug package to `0.1.0.r9.g3c5f935-1`.
Generated non-root `.SRCINFO`, checksum verification, `namcap PKGBUILD`, Go
`check()`, installed ownership/modes, passthrough, guard failure, and review
layout passed. `pacman -Q auroscope` reported `auroscope 0.1.0.r9.g3c5f935-1`.

The package gate first reproduced #60 on the old installed r8 artifact with
an unreadable `hermes-agent-desktop/pkg` directory. After the upgrade, bare
`/usr/bin/auroscope` completed with deterministic Paru responses for the official
update and empty AUR query, without invoking Codex. It left the artifact mode
at `000` and set only the clone root to `700`.

The full pinned-Paru gate passed against `/usr/bin/auroscope`: native
selection/color behavior, real AUR acquisition/build/install, edit and re-audit,
exact drift refusal, explicit retry with a second audit and approval, skip,
and official installs without a Codex call. The original container completed
successfully (Docker exit code 0); its complete logs were recovered after the
calling agent timed out. Codex was a deterministic test executable; these runs
do not claim a new real-model qualification.

This candidate requires human review and merge before a plain `git pull` on
`main` obtains it. Preserve the pinned source commit in merge history rather
than squashing it away or deleting its only reachable branch. No desktop
installation was changed by these tests.

## Keeping the package current

Follow the delivery rules in the root `AGENTS.md`. Before declaring a change
ready, run:

```console
bash packaging/aur/scripts/check-freshness.sh
packaging/aur/scripts/test-package.sh
AUROSCOPE_E2E_SUPPORTED_PARU=1 scripts/e2e-supported-paru.sh
```

Run these commands from the repository root. Both package-verification entry
points invoke the freshness check before starting Docker; source-only E2E mode
remains explicitly separate. The check compares code, tests, test scripts,
Go dependencies, and the installed root README with the immutable source pin.
It rejects changed tracked inputs and new non-ignored source files. Keep its
input list in sync with the recipe. It is a local release gate, not a configured
server-side branch protection or an automatic release publisher.

Commit source changes first, then advance `_commit`, the `rN` revision in
`pkgver`, its commit suffix, and `sha256sums` together. Generate `.SRCINFO` with
Arch `makepkg --printsrcinfo`. Recipe-only rebuilds increment `pkgrel`.
A new package identity is required whenever delivered contents change; forcing
a rebuild of an old pin is not an update. The package test keeps the actual
previous package archives present to exercise this upgrade path every time.

After merging a package update, from an existing clone:

```console
git pull --ff-only
cd packaging/aur
makepkg -si
pacman -Q auroscope
```

If already in `packaging/aur`, omit the `cd` command. `auroscope --version`
intentionally passes through to Paru; use `pacman -Q auroscope` for this package.
The installed README comes from the immutable source and may retain historical
candidate wording; current recipe metadata identifies the delivered snapshot.

Upstream has not yet declared a software license. `LicenseRef-Unspecified` records that fact; it must be replaced when upstream adopts a license.
