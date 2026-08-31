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

## Disposable verification

- Built the exact tag in `archlinux:base-devel`: locked build failed against libalpm 16 as expected; applying the AUR recipe's `cargo update alpm alpm-utils` produced `paru v2.1.0 - libalpm v16.0.1`.
- With `v2.1.0`, `printf '1\n' | paru -Ssaq --interactive paru` produced the menu and `paru` together on stdout, stderr empty, exit `1`.
- Built upstream `master` commit `9ac3578`; the same command produced only `paru` on stdout, menu on stderr, exit `1`.
- `paru -P --order paru` at `9ac3578` emitted `REPO ...` and `AUR ...`; with synced databases it exited `0`, while absent repository metadata produced `MISSING ...` records and exit `1`.
- A benign local `paru -B` with layered config confirmed the final `PreBuildCommand` override, but Paru generated `.SRCINFO` through makepkg before that hook.
- A configured marker-writing PKGBUILD repository with `PkgbuildsOnly` and `GenerateSrcinfo` remained unexecuted when trusted CLI flags `--repo --mode=aur` were injected; the marker was absent, confirming config mode was reset then limited to repo+AUR on commit `9ac3578`.

## Project rule

The durable adapter must not parse Paru's human menu and requires a stable release containing `d1dfbc4`, plus executable capability tests. [ADR-0002](../decisions/0002-paru-native-selection-compatibility.md) permits a strictly temporary exception for Paru 2.1.0: an isolated adapter may parse only the verified combined stdout format, must reject ambiguity or format/locale drift, and must be removed after a fixed stable release passes the capability matrix. Treat exit `1` as expected only for the exact verified interactive search-selection subprocess when valid selected targets were recovered and the child was not terminated by a signal. For intercepted v1 flows, reject modes containing pkgbuilds and local/path-like targets, then normalize repo-only, AUR-only, or combined intent with final trusted reset flags; reject local/PKGBUILD-repository builds until inspection can precede every makepkg invocation.

## Sources consulted

- `man/paru.8`, `man/paru.conf.5`
- `src/lib.rs`, `src/install.rs`, `src/util.rs`, `src/exec.rs`
- AUR `paru/PKGBUILD`
- commits `70f66dc9`, `d1dfbc48`, `9ac35788`
