# Dependency note — Paru 2.1.0 and post-2.1 integration contract

## Context

AURoscope must preserve Paru's native selection and resolver while obtaining a machine-readable selected target list and enforcing an exact-recipe guard before build.

## Verified baseline

- Stable upstream tag: `v2.1.0`, commit `70f66dc9eddb40e264ee6c9197541262b7792c9c`.
- Current AUR recipe inspected: `paru 2.1.0-2`, AUR commit `329be2113c590046cb29858c23d9b96a8d7bd586`.
- Current upstream `master` inspected: `9ac3578807a87858651e81a02586ceb947686e7c`.
- Current Arch image used by spikes: Pacman `7.1.0`, libalpm `16.0.1`.

## Verified findings

1. Bare Paru becomes `-Syu`; bare search terms become sync interactive selection.
2. `paru -Ssaq --interactive TERM` is intended to display Paru's menu and emit selected names for another tool. The sync-search path returns exit status `1` even when a package was selected.
3. Released `v2.1.0` contains a file-descriptor redirection bug: menu text and the selected name both remain on stdout. This makes robust selection capture impossible without parsing human UI. Commit `d1dfbc48895a2a3b3fadc14569c3f2ffb5112b2e` fixes the redirection; no stable tag contains it at the time of inspection.
4. A current AUR-style build of `v2.1.0` must run `cargo update alpm alpm-utils` to support libalpm 16. Building the unmodified locked tag against Pacman 7.1 fails because `alpm 4.0.3` supports libalpm 15 only.
5. `PreBuildCommand` runs via `sh -c`, with `PKGBASE`, `VERSION`, and cwd set to the recipe directory. It runs for each planned package base even if Paru later reuses an already-built package.
6. In `v2.1.0`, all pre-build commands run after recipe download but before Paru's own review. Therefore AURoscope must invoke execution with Paru review disabled after AURoscope's review; otherwise Paru's file-manager review can mutate the recipe after the guard.
7. Paru may reuse a same-version package artifact from cache after the guard. Recipe approval alone does not authenticate that binary artifact.
8. `paru -P --order` is not a stable machine protocol today: `v2.1.0` documents/prints `INSTALL ...` repository records, while post-release `master` commit `9ac3578` prints `REPO ...` from `src/order.rs` and its man page still documents `INSTALL`. The adapter must be version/capability-gated and reject unknown records.
9. `paru -B` calls makepkg `--printsrcinfo` while constructing its local PKGBUILD repository, before `PreBuildCommand`. Configured PKGBUILD repositories may likewise generate `.SRCINFO` when absent or forced. Paru also special-cases any sync target beginning `./` before normal mode handling. V1 must reject local/path-like targets before starting Paru and reset config mode with trusted CLI flags.
10. A generated Paru config can include the original effective config and then override `PreBuildCommand` in a final `[bin]` section; a disposable build spike confirmed the later command won.
11. Pacman 7.1 cannot replace installed `paru 2.1.0` with the absent AUR-only `paru-git` provider while installing an AURoscope package that both depends on `paru` and conflicts with `paru<=2.1.0`: accepting removal leaves the dependency unsatisfied and aborts the transaction. The provider replacement must be a separate AUR-helper transaction. The self-hosted recipe checks this exact installed package in `prepare()` so `makepkg -si` fails before the Go build with an actionable remediation, while package metadata retains the conflict as the final installation guard.

## Disposable verification

- Built the exact tag in `archlinux:base-devel`: locked build failed against libalpm 16 as expected; applying the AUR recipe's `cargo update alpm alpm-utils` produced `paru v2.1.0 - libalpm v16.0.1`.
- With `v2.1.0`, `printf '1\n' | paru -Ssaq --interactive paru` produced the menu and `paru` together on stdout, stderr empty, exit `1`.
- Built upstream `master` commit `9ac3578`; the same command produced only `paru` on stdout, menu on stderr, exit `1`.
- `paru -P --order paru` at `9ac3578` emitted `REPO ...` and `AUR ...`; with synced databases it exited `0`, while absent repository metadata produced `MISSING ...` records and exit `1`.
- A benign local `paru -B` with layered config confirmed the final `PreBuildCommand` override, but Paru generated `.SRCINFO` through makepkg before that hook.
- A configured marker-writing PKGBUILD repository with `PkgbuildsOnly` and `GenerateSrcinfo` remained unexecuted when trusted CLI flags `--repo --mode=aur` were injected; the marker was absent, confirming config mode was reset then limited to repo+AUR on commit `9ac3578`.
- In the pinned Arch package gate, an installed disposable `paru 2.1.0-1` satisfied `depends=('paru')`; the AURoscope recipe then rejected it during `prepare()` before creating an archive. After a separate replacement with a disposable `paru-git` package providing `paru`, the same recipe built and installed successfully.

## Current use under ADR-0015

### Empty AUR updates and native warnings

At the pinned `9ac3578807a87858651e81a02586ceb947686e7c`,
`src/query.rs::print_upgrade_list` returns 1 when it prints no upgrades.
An installed package missing from AUR contributes no target; an out-of-date
flag does not by itself imply an available version upgrade. The previous
adapter rejected this normal empty result as a child failure (issue #84).

Status 1 with empty stdout is not sufficient proof of success: a network or
runtime failure can have the same shape. The candidate lets native Paru
confirm an empty result with `-Su --mode=aur --skipreview`, using the original
terminal streams. Paru prints its own missing/out-of-date warnings and
"nothing to do" text; AURoscope does not parse or reconstruct that UI.
Native failure and interruption remain failures. No second official upgrade
or configured local PKGBUILD repository is entered by this AUR-only command.

The confirmation uses the existing transaction hook with an empty approval
list. A newly discovered recipe is refused before its code executes, rather
than using a stale empty query as permission to build. This conservative race
case requires a fresh invocation to query and audit the newly available update.
Updates already found by the query retain the existing audited target flow.

Verified in disposable Arch: `TestEmptyUpdateRealParu` supplies a private
libalpm database and local RPC response to the actual pinned Paru. The empty
query returns 1; the corrected flow returns 0 and retains both native warnings.
`TestEmptyAURUpdate` checks native streams and input, nonzero statuses, partial
failed output, the empty approval guard, and absence of provider configuration.
The full source-only supported-Paru gate also passed, including real build,
edit/re-audit, drift refusal, retry and audit-free official installation.

Source evidence: `build/issue-84/source-paru.log`. The subsequent r23 package
pins `ee992223ee02dea214b5bb2c162c614ff250edd4`. Both package gates exited 0;
the r22-to-r23 upgrade retained the old archives without forcing a rebuild.
The same regression tests passed against installed `/usr/bin/auroscope`,
including actual pinned Paru and both native warnings. Package and installed
integration evidence: `build/issue-84/package-r23.log` and
`build/issue-84/paru-r23.log`. No workstation installation was performed.

These findings are historical evidence, not a complete active adapter contract. The first executable task must revalidate one supported Paru version and prove the exact native search/selection, AUR worktree, edit/re-audit, skip/exclusion, final build, and `PreBuildCommand` path required by [ADR-0015](../decisions/0015-minimal-llm-first-wrapper.md). Do not carry forward the transitional human-menu parser, broad mode classifier, or closure machinery merely because they were explored here.

## Sources consulted

- `man/paru.8`, `man/paru.conf.5`
- `src/lib.rs`, `src/install.rs`, `src/util.rs`, `src/exec.rs`
- AUR `paru/PKGBUILD`
- commits `70f66dc9`, `d1dfbc48`, `9ac35788`
