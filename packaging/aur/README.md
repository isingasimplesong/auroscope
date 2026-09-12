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

For archives requiring authentication, prefetch the real source archives on the
host into `packaging/aur/cache/sources/`, using the filenames declared by each
recipe. Include the current archive and the previous one referenced by its
embedded recipe. Use the operator's own credentials; never copy credentials or
host makepkg configuration into this directory. It is ignored by Git.

The package gate copies these archives into disposable Arch and sets a shared
`SRCDEST` for both builds. Each recipe still verifies its own SHA-256. Missing
archives use makepkg's normal download path; inaccessible sources remain errors.
The normal `scripts/test-package.sh` entry point needs no PATH shim or environment
override once the cache is populated. If acquisition fails before the stable-Paru
check, the gate now prints makepkg's actual failure rather than hiding it behind
a missing expected diagnostic.

## Upstream snapshot

### Recovery checkpoint pending

The retained r12 results below qualify the earlier source, not the current
working tree. The subsequent wrapper run passed the package gate but failed
in the supported-Paru gate while anonymously fetching the private archive
(HTTP 404, `test-64.log` lines 926-936).

The supported-Paru script now consumes the same prefetched archive directory
as the package gate, with checksum verification and no Docker PATH shim.
This changes a tracked test input under `scripts/`: the wrapper must preserve
a new immutable source checkpoint before the recipe can advance. The retained
r12 recipe is not fresh for this change. New metadata, checksum and full
installed-artifact validation remain pending; this is not a ready release.

### Retained r12 verification

Candidate package `0.1.0.r12.g78ad044-1` adds the editable audit prompt from #64
to the shared provider configuration. A missing file receives the complete
default prompt; existing files and customized prompts survive upgrades and audits.
It preserves #63's complete auxiliary recipe context and #68's providers.
The default-model change in #65 remains separate retained work, not delivered here.

The immutable source was read back on remote `loop/subject-64`:
`78ad044b5e627ea98638f0aa2074f42445c4e63b`

The authenticated archive download has SHA-256:
`bb1789647dddd3baa3b58ce806dd3610037b5c218eadc32653719a7053156197`

Non-root Arch `.SRCINFO` generation, freshness, Go tests/vet and both disposable
Arch gates passed. The package gate upgraded r11 with its real archives present,
without `--force`, and confirmed that Codex received the preserved custom prompt.
Both gates reported:

```text
auroscope 0.1.0.r12.g78ad044-1
```

The full supported-Paru gate exercised `/usr/bin/auroscope`: prompt creation,
customization and preservation, auxiliary scripts, provider success/failure,
native colors, real AUR build/install, edit/re-audit, drift refusal, explicit retry,
skip and audit-free official operations passed. Providers were deterministic;
this does not qualify a new live model or change a desktop installation.

Anonymous archive downloads returned HTTP 404 for both r11 and r12 during this
run. The gates therefore used the real archives downloaded with profile credentials
in a shared makepkg `SRCDEST` cache; makepkg still verified their recipe checksums.
No credentials entered the containers. An uncached anonymous bootstrap is not
verified in this access state; authenticated source acquisition is required.

The subsequent wrapper failure was not a prompt regression: its normal package
test lacked the earlier run's Docker cache shim. The repaired package gate was
rerun without that shim, using fresh authenticated r11/r12 archives in the
documented cache. It passed generation and byte comparison of non-root Arch
`.SRCINFO`, checksums, stable-Paru refusal, r11-to-r12 upgrade without `--force`,
and transmission and preservation of the custom prompt. Its process exited 0.

The full installed-artifact supported-Paru gate also exited 0 on this same pin,
including `TestPromptConfigurationLifecycle` and the auxiliary/provider tests.
That unchanged integration script still used the retained archive-cache Docker
shim; its anonymous download path is not newly qualified. Local evidence:
`logs/recovery-64/package-normal.log` and
`logs/recovery-64/paru-revalidated.log`. Both again reported
`auroscope 0.1.0.r12.g78ad044-1`.

Only the external package-test harness changed in this recovery, not an input
consumed by the recipe's build, check or package functions. The existing remote
source pin and candidate package identity therefore remain valid.

The final fetch still found `main` at `23f64e1`, fully included in this source.
Preserve this pin and earlier pins in remote history. Wrapper publication and
human review/merge remain separate from these successful package validations.

### Previous r11 verification

Candidate package `0.1.0.r11.gad96e16-1` delivers the explicit audit-provider
selection authorized by #68: Codex remains the default, with Claude Code,
Anthropic API, OpenAI API and configurable OpenAI-compatible API alternatives.
Configuration and bounded qualification evidence are documented in the root
README and `docs/dependency-notes/audit-providers.md`.

The immutable source pin, verified on remote `loop/subject-68`, is:
`ad96e164679bbdd0a2b474d9712f238cc3a06679`

The downloaded archive SHA-256 is:
`ba45ebc42831195e9e474fc9cc715978c205eb888d0b622ff2487e7c43f5fde1`

Non-root Arch `.SRCINFO` generation, freshness, Go tests/vet and both disposable
Arch gates passed. The upgrade test retained the real r10 archives and installed
r11 without `--force`. Both gates reported:

```text
auroscope 0.1.0.r11.gad96e16-1
```

The supported-Paru gate exercised `/usr/bin/auroscope`, including configured HTTP
approval and failed-audit skip, explicit Claude selection, retry/cancel/skip with
no Codex fallback, and complete auxiliary-script audit input. Native colors,
real AUR build/install, edit/re-audit, identity drift refusal, explicit recovery
and audit-free official operations also passed. These deterministic provider
tests do not qualify a live API endpoint/model or native Claude authentication.

The final fetch confirmed that the source includes current `origin/main` at
`da9dca7`; no concurrent source fix is omitted. Preserve the source commit in
remote merge history. Publication and human review/merge remain pending; no
desktop installation was changed.

### Previous r10 verification

Candidate package `0.1.0.r10.g04025fc-1` pins immutable AURoscope commit
`04025fceaaa0df59d357275f4f337b6b6c9693e7`, verified on the remote issue branch.
It retains r9's fixes and supplies every tracked recipe file to each audit,
including unchanged installation, removal and helper scripts. The diff remains
additional context. Binary files retain metadata and hashes; downloaded upstream
sources and untracked build leftovers remain outside this recipe audit.
The real archive SHA-256 is:
`d3774042b9466e2eb03af00acfff3b2fd91e449643b7293681815773d369c5d8`

The disposable Arch package gate passed: non-root generated `.SRCINFO`, archive
checksum, `namcap`, Go tests, and upgrade from r9 with its archives still present,
without `--force`. `pacman -Q auroscope` returned:

```text
auroscope 0.1.0.r10.g04025fc-1
```

The full supported-Paru gate also passed against `/usr/bin/auroscope`.
`TestAuditIncludesAuxiliaryScripts` verified full, differential and unchanged
audits, complete auxiliary text and no execution during collection. The gate
also retained native selection/colors, real AUR build/install, edit/re-audit,
drift refusal, explicit retry, skip and audit-free official installations.
Codex was deterministic in these tests; this is not a new real-model qualification.

The final remote check found `main` at
`1b70f8c67e5afe9239fbccdac9738d942c61e7fc`. Its additional commits change only
unpackaged provider-decision documentation, not shipped inputs. Publication must
preserve those additions and the immutable source pin. Human review and merge
remain required; no desktop installation was changed.

### Previous r9 verification

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
