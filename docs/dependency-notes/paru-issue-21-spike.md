# Dependency note — Paru issue #21 disposable spike

## Scope and pinned surface

Issue [#21](https://git.2027a.net/2027a/auroscope/issues/21) was exercised in a disposable `archlinux:base-devel` container pinned by image digest `sha256:68bfc3b0d277b08a99101dc9b94aaa03e5ae70cf1b4fb965c03b2b87b915760d`.

The supported spike surface is Paru commit [`9ac3578807a87858651e81a02586ceb947686e7c`](https://github.com/Morganamilo/paru/commit/9ac3578807a87858651e81a02586ceb947686e7c), described by Git as `v2.1.0-67-g9ac3578`, with Pacman `7.1.0` and libalpm `16.0.1`.

The tagged Paru `v2.1.0` is **not** this machine contract: its interactive-search redirection bug mixes menu text with selected names. The pinned commit includes fix [`d1dfbc4`](https://github.com/Morganamilo/paru/commit/d1dfbc48895a2a3b3fadc14569c3f2ffb5112b2e). Production must reject an unsupported Paru surface rather than parse human menu output.

## Upstream source consulted

- [`src/lib.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/lib.rs#L329-L353): native interactive search writes selected targets after the menu and deliberately returns `1`.
- [`src/order.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/order.rs#L8-L32): `-P --order` performs resolution and emits `REPO`, `AUR`, `SRCINFO`, conflict, and missing records.
- [`src/download.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/download.rs#L177-L220): `-G` separates repository and AUR targets and acquires AUR PKGBUILDs into the current directory.
- [`aur-fetch::make_view`](https://github.com/Morganamilo/aur-fetch.rs/blob/128a15fce9391303489b7a237e10f4f016fd17fb/src/fetch.rs#L360-L404) and its [fetch/merge path](https://github.com/Morganamilo/aur-fetch.rs/blob/128a15fce9391303489b7a237e10f4f016fd17fb/src/fetch.rs#L407-L448): review points at the clone; a later fetch discards uncommitted changes but rebases local commits, matching the two-invocation edit contract proved below.
- [`src/install.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/install.rs#L1123-L1171): Paru downloads worktrees, invokes every `PreBuildCommand`, then performs its own review unless `--skipreview` is set.
- [`src/install.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/install.rs#L1568-L1578) and [`man/paru.conf.5`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/man/paru.conf.5#L383-L391): the hook runs through `sh -c` in the package worktree with `PKGBASE` and `VERSION`.
- [`man/paru.8`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/man/paru.8#L246-L257) and [`src/install.rs`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/install.rs#L899-L945): `--repo` restricts targets/sysupgrade to official repositories and forwards that path early to Pacman.

## Reproducible path and observed evidence

The container built the pinned Paru commit, configured `CloneDir = /home/builder/aur`, and used `hello` as a disposable AUR recipe. The recipe was treated as hostile data and was not sourced during acquisition or audit. A benign top-level marker was added only to prove when package code first executed.

### Search, selection, and origin

```console
$ printf '1\n' | paru -Ssaq --interactive hello >selected 2>menu
$ printf '%s\n' $?
1
$ cat selected
hello
$ paru -Si hello | grep -E '^(Repository|Name)'
Repository      : aur
Name            : hello
```

The menu remained on stderr, the selected package alone was on stdout, and status `1` is the documented behavior of this Paru path rather than selection failure.

### Exact worktree, edit, re-audit identity, and final build

```console
$ cd /home/builder/aur && paru -G hello
$ realpath hello
/home/builder/aur/hello
$ git -C hello rev-parse HEAD
51cec6333515471681ec8aa00943145d420311fa
$ sha256sum hello/PKGBUILD
# d805cbe191989ff5098c164d655706151c55c04633e18edf0ab1d7a36ed00ddf
```

Acquisition left the recipe marker absent: no package-supplied code had run. After the disposable edit, the worktree change was committed locally and re-audited as identity:

```text
commit  fe82dad2b10c09f7cf0480b827e8098f827da6a7
sha256  ab328f1e7697b4c9e7c0fe4a9de496629f6329b15a3dded2868d1efc492cd46b
```

The final command was:

```console
$ paru -S aur/hello --skipreview --noconfirm
```

It exited `0`, printed `PKGBUILDs up to date`, built and installed `hello 2.12.1-2`, and retained the same path, local commit, and PKGBUILD digest. The hook observed:

```text
PKGBASE=hello
VERSION=2.12.1-2
PWD=/home/builder/aur/hello
PKGBUILD_SHA256=ab328f1e7697b4c9e7c0fe4a9de496629f6329b15a3dded2868d1efc492cd46b
EXPECTED_SHA256=ab328f1e7697b4c9e7c0fe4a9de496629f6329b15a3dded2868d1efc492cd46b
RECIPE_MARKER_BEFORE_GUARD=absent
```

Only after the hook returned did makepkg source the PKGBUILD; the marker then contained `sourced-from:/home/builder/aur/hello`.

A second run introduced committed post-approval drift and used `--rebuild`. The hook observed digest `606252672ce831911e67fc9fb4c3b79ad7a002faa5d507d09bc49fe4c2930424`, rejected it against the approved digest, Paru exited `1`, and the recipe marker remained absent.

**Edit decision:** keep `edit + re-audit` in v1. It is proved only when AURoscope snapshots the edited cache worktree as a local Git commit before re-audit; uncommitted Paru review edits are not a supported handoff. Final execution must reuse the configured clone directory and pass `--skipreview`.

### Skip followed by fresh Paru resolution

```console
$ paru -P --order hello archlinux-hello
AUR TARGET hello hello
AUR TARGET archlinux-hello archlinux-hello
$ paru -P --order archlinux-hello
AUR TARGET archlinux-hello archlinux-hello
```

Both commands exited `0`. Omitting skipped `hello` from the second invocation removed it while Paru recomputed all repository dependencies. AURoscope therefore passes approved targets, not a stored dependency closure; Paru remains authoritative and may reject a remaining target whose dependency was skipped.

### Official operations are transparent

```console
$ paru -Syu --repo --noconfirm
:: Starting full system upgrade...
 there is nothing to do
$ paru -S --repo tree --noconfirm
# native Pacman transaction; installed tree 2.3.2-1
```

Both commands exited `0`. The complete `PreBuildCommand` log had the same SHA-256 before and after both operations, proving that official-only paths never entered the AUR hook/audit boundary.

## Implementation contract from this spike

1. Require a tested Paru surface that has the interactive stdout/stderr fix and the `REPO`/`AUR` order records; reject anything else.
2. Preserve Paru's search menu, capture only its selected names, and interpret that command's status `1` according to the pinned contract.
3. Use `-P --order` only as Paru's machine resolution surface and `-G` to acquire each resolved AUR package base into the configured clone directory.
4. Read and hash worktree bytes without invoking makepkg or sourcing recipe files.
5. Commit an edited worktree locally, recompute its complete identity, and rerun Codex before approval.
6. Relaunch Paru with approved targets only, a transaction-private `PreBuildCommand`, and `--skipreview`; the hook rejects unknown or changed worktrees.
7. Let Paru perform the fresh final resolution, makepkg build, and Pacman installation. Official-only commands bypass all audit machinery.
