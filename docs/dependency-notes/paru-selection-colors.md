# Paru native selection colors

## Exact surface and cause

Issue [#34](https://git.2027a.net/2027a/auroscope/issues/34), comments 1819 and 2035, reports working search but missing colors. Paru commit `9ac3578807a87858651e81a02586ceb947686e7c` chooses automatic colors before its interactive-search redirection:

- [`src/config.rs:104-111`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/config.rs#L104-L111): `auto` requires both stdout and stderr to be terminals.
- [`src/config.rs:719-724`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/config.rs#L719-L724): Pacman's `Color` setting enables this automatic mode in the absence of an explicit override.
- [`src/lib.rs:342-350`](https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/lib.rs#L342-L350): `-Ssaq --interactive` redirects the native menu to stderr, restores stdout for targets, and returns status 1.

The previous Go buffer capture made child stdout a pipe. Therefore even a native color-enabled terminal lost colors on stderr.

## Narrow correction

Only for interactive selection, and only when both caller output descriptors are actual terminals, capture machine stdout through a private PTY. Keep stdin and menu stderr on their original descriptors; keep the current process-group/foreground handling. Copy the initial stdout geometry. Drain capture concurrently, close the slave after reaping the child (including start failure), and treat Linux PTY `EIO` as end-of-stream.

This uses the already-pinned `github.com/creack/pty v1.1.24` (`Open`, `GetsizeFull`, `Setsize`); it adds no dependency. CRLF from the slave is handled by the existing line parser. Targets remain validated, not stripped of arbitrary ANSI. There is no human-menu parser, force-color flag, replacement Pacman configuration parser, or change to official transactions, order records, acquisition, or final builds.

Do not unconditionally add `--color always`: that would override a user's disabled `Color` setting and could color redirected output. The PTY preserves Paru's own decision instead.

## Verification

- TDD: `TestSelectionPreservesNativeColorDetection/terminal` failed before the fix with `native color = false`; the matrix is green after it.
- Deterministic tests cover terminal/redirected combinations, exact target capture with no leakage, exec failure, actual controlling-terminal input/foreground restoration, and SIGINT capture termination.
- `CGO_ENABLED=1 go test ./... -count=1`, `go vet ./...`, and the focused selection race test pass.
- The pinned Arch gate's `TestSelectionRealParuColors` uses real Paru with generated Pacman configs. Color-enabled terminal, Color-disabled terminal, and redirected output all pass; each selects the exact `hello` target and preserves the native prompt.
- The selection-only gate was executed successfully with:

  ```console
  AUROSCOPE_E2E_SUPPORTED_PARU=1 AUROSCOPE_E2E_SELECTION_ONLY=1 scripts/e2e-supported-paru.sh
  ```

  It exits after real search/selection tests, before recipe acquisition or execution. Docker package-manager activity stays in the disposable container. The default gate still continues to the pre-existing build/install scenarios; do not run it where recipe execution is prohibited.

## Installed artifact verification

The operator completed delivery verification after Mathieu explicitly authorized
the disposable recipe gates in #34 comment 2189 and #45 comment 2187.
Package `0.1.0.r6.gf97d6ab-1` combines the color fix with the merged Codex admission
change. The pinned source is `f97d6ab75fbcacf60f17cd02649df6c145795e25`.

- The installed old `0.1.0.r5.gd42be9f-1` reproduced missing terminal colors.
  That first probe also had an incorrect expected exit status: order failures are
  wrapped as status 1. The probe was corrected to check status 1 and the specific
  order-status-73 diagnostic, proving the exact selected target reached planning.
  The additional status assertions in the old log are harness failures, not bugs
  in the package; the observed missing ANSI colors remain the regression evidence.
- The corrected matrix passes for the installed `/usr/bin/auroscope` with real
  pinned Paru: colors enabled, colors disabled, and output redirected. The native
  prompt remains visible and the exact selected `hello` target reaches planning.
- The package build/install gate passes with fake Codex 0.153.4. The full real-Paru
  gate uses the installed artifact for subsequent build/install, edit/re-audit,
  skip and official-only checks; the fake-Paru smoke also passes.
- Go tests, focused selection race tests, vet, shell syntax, checksum, generated
  `.SRCINFO` and namcap checks pass. No host package transaction was performed.

The historical drift scenario accepts any nonzero exit; its successful gate run
is not sufficient proof of the rejection cause. That test-hardening follow-up and
an independent official-only search bug are recorded in #50, not silently included
in this colors fix. PR #49 still requires human review and merge; no workstation
upgrade or AUR publication is claimed.
