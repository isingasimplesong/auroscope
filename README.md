# AURoscope

AURoscope is a small terminal wrapper around [Paru](https://github.com/Morganamilo/paru) and Pacman. Its purpose is narrow: before Paru builds an AUR recipe, AURoscope asks Codex CLI to audit the exact recipe change, shows the result to the user, and records a per-package decision.

## Intended interface

```console
auroscope                    # native official update, then audited AUR updates
auroscope <search terms>     # Paru-native search and numbered selection
auroscope -S <packages>      # explicit installation
auroscope <other Paru args>  # transparent passthrough when no AUR build occurs
```

## Product flow

1. Official repository updates and installs remain native Paru/Pacman operations. They receive no AURoscope audit.
2. Paru identifies the AUR package bases that would be built.
3. AURoscope compares each exact recipe with the last successfully approved commit, or sends the full recipe on first use.
4. Codex CLI returns a structured audit of the packaging diff and relevant recipe files.
5. AURoscope shows a concise assessment and packaging-risk level; the full findings and diff appear only when the user chooses `inspect`.
6. The user chooses by number, initial, or full word: `approve`, `inspect`, `edit` and re-audit, `skip`, or `cancel` per AUR package base.
7. AURoscope relaunches Paru with the official targets and approved AUR targets. Paru resolves dependencies, calls `makepkg`, and installs through Pacman normally.
8. A minimal `PreBuildCommand` rejects a recipe whose identity differs from the audited one.

AURoscope does not replace Paru's search UI, resolver, build machinery, or Pacman. It does not audit official packages.

## V1 shape

- Go executable for Arch Linux `linux/amd64`.
- Codex CLI is the only LLM backend.
- SQLite stores only the last successful recipe baseline and audit history.
- One internal Go package initially; split only when demonstrated behavior requires it.
- No deterministic rule engine, plugin system, build sandbox, cached artifact reuse, or same-UID security protocol.

The accepted architecture is in [`docs/architecture/minimal-v1.md`](docs/architecture/minimal-v1.md) and [`ADR-0015`](docs/decisions/0015-minimal-llm-first-wrapper.md). A narrow disposable-Arch Paru worktree spike must be completed before the new implementation plan is written.

## Current implementation

The issue #24 implementation provides the minimal v1 wrapper in `cmd/auroscope` and `internal/app`. The issue #45 candidate implements [ADR-0046](docs/decisions/0046-codex-minimum-version-without-ceiling.md): an inclusive Codex CLI minimum of `0.150.1` without a ceiling, including future stable major versions. It admits only the strict stable banner `codex-cli MAJOR.MINOR.PATCH` (optionally terminated by LF or CRLF), compared numerically; old versions, prereleases, build suffixes, and malformed banners are rejected. Admission is not qualification: the real `0.153.4` production-client audit passed after the authorized host authentication was selected, and a separate direct sandbox probe demonstrated read access and OS-denied workspace writes. These bounded observations do not establish all model-driven sandbox semantics; see the [qualification evidence and limits](docs/dependency-notes/codex-cli-contract.md). The candidate package `0.1.0.r5.gd42be9f-1` pins this corrected implementation and passed build/install plus installed 0.153.4 admission checks in disposable Arch. The separate real-Paru gate passed too. PR #48 remains subject to human merge; an existing installation is not upgraded by these tests. The unmerged selection-colour follow-up in PR #49 is separate and is not included in this snapshot.

## Install on Arch Linux

The self-hosted AUR-style recipe lives in [`packaging/aur`](packaging/aur). Publication on `aur.archlinux.org` remains deferred.

```console
# Install Codex by any supported method, for example:
npm install -g @openai/codex

# Install the currently tested compatible provider. AURoscope needs Paru's
# post-2.1 interactive-output fix and machine order records.
paru -S paru-git

# Then bootstrap AURoscope without asking it to audit itself:
git clone https://git.2027a.net/2027a/auroscope.git
cd auroscope/packaging/aur
makepkg -si
```

Stable Paru 2.1.0 is not compatible: its interactive search writes the human menu and selected targets to the same stream, so AURoscope cannot recover the selection without parsing localized UI. The package depends on the virtual `paru` capability so an installed compatible provider such as `paru-git` satisfies `makepkg`, but it conflicts with the known-incompatible stable package version `paru<=2.1.0` instead of allowing installation to produce a silent search prompt. Pacman still cannot fetch an absent AUR provider, so install `paru-git` before bootstrapping AURoscope. The currently tested Paru surface is commit `9ac3578807a87858651e81a02586ceb947686e7c`. AURoscope validates the required selection and order behavior when those paths run and fails closed on incompatible output; a package name or `paru --version` string alone cannot prove that post-release contract. Codex is deliberately not a Pacman dependency: AURoscope uses the `codex` executable found on `PATH`, whether it came from npm, an Arch package, or another installation method. `codex --version` must report a supported version, and Codex must be authenticated for the user who runs AURoscope. During the first desktop trial, invoke `auroscope` explicitly rather than replacing `paru` with an alias.

## Development verification

```console
CGO_ENABLED=1 go test ./...
go vet ./...
packaging/aur/scripts/test-package.sh
AUROSCOPE_E2E_DISPOSABLE_ARCH=1 scripts/e2e-disposable-arch.sh
AUROSCOPE_E2E_SUPPORTED_PARU=1 scripts/e2e-supported-paru.sh
```

The package test and both integration scripts require Docker. The package test builds and installs `packaging/aur/PKGBUILD` in disposable Arch. The first integration script is a fast fake-Paru/fake-Codex smoke. The supported-version gate builds pinned Paru commit `9ac3578807a87858651e81a02586ceb947686e7c`, uses real AUR acquisition/resolution/build and Pacman installation only inside the disposable container, proves the real guard and drift rejection, and confirms that official-only work invokes no Codex audit.

## Previous design

The earlier architecture was intentionally superseded because it made the LLM optional and accumulated resolver, approval, state, recovery, and test machinery outside the product's purpose. Historical material remains under [`docs/archive/pre-llm-first/`](docs/archive/pre-llm-first/).
