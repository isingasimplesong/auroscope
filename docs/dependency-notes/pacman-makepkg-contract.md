# Dependency note — Pacman and makepkg 7.1 contract

## Context

AURoscope must preserve official Pacman transactions and inspect recipes without executing or sourcing their PKGBUILDs.

## Verified baseline

- Upstream Pacman tag: `v7.1.0`, commit `5683f8477a0afcc6b331766175a83445b2dcfe89`.
- Current Arch package inspected: `7.1.0.r9.g54d9411-2`; packaging commit `abdc0dfedf3ca553a02dde8551e972fe745535b7`, patch-level source commit `54d94116164b0b2202c6061c4a59c6f3e70820d8`.
- The nine post-release commits do not change the Pacman print contract or the makepkg PKGBUILD-sourcing boundary relevant here.

## Verified findings

1. Pacman `-S --print --print-format` performs dependency resolution without committing the transaction and can emit repository, name, version, pkgbase, URL and checksums. It covers repository packages, not AUR build recipes.
2. `-Syu` is the supported complete repository refresh plus system upgrade. Refreshing with `-Sy` and then withholding repository upgrades creates the unsupported partial-upgrade hazard.
3. Pacman's local database lock is authoritative for Pacman transactions; AURoscope must not invent a competing lock or remove `db.lck`.
4. `makepkg --printsrcinfo`, `--packagelist`, and `--verifysource` all source the PKGBUILD. They are execution boundaries and must never be used by AURoscope's inspection collector.
5. `PKGDEST`, `SRCDEST`, `SRCPKGDEST`, `LOGDEST`, and `BUILDDIR` redirect makepkg outputs/work, but do not make recipe evaluation safe.
6. makepkg documents stable exit categories `0..17`; Paru can collapse or transform failures, so AURoscope must preserve raw child status/signal and add its own stable wrapper category rather than pretending every nonzero is equivalent.

## Disposable verification

In `archlinux:base-devel` with Pacman/makepkg `7.1.0`, a PKGBUILD containing a top-level marker write was tested separately with:

- `makepkg --printsrcinfo` → exit `0`, marker written;
- `makepkg --packagelist` → exit `0`, marker written;
- `makepkg --verifysource` → exit `0`, marker written.

No production or repository code was produced by this spike.

## Project rule

Read tracked files and committed `.SRCINFO` as hostile bytes. Never invoke makepkg, Bash, or `source` to derive inspection metadata. Use Paru's resolver/AUR RPC for planning and let makepkg execute only after an exact human approval passes the pre-build guard.

## Sources consulted

- `doc/pacman.8.asciidoc`
- `doc/pacman.conf.5.asciidoc`
- `doc/makepkg.8.asciidoc`
- `doc/makepkg.conf.5.asciidoc`
- `scripts/makepkg.sh.in`
- `src/pacman/sighandler.c`, `src/pacman/util.c`
- Pacman test suite under `test/pacman/tests/`
- Arch package `PKGBUILD`
