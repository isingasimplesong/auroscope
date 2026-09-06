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

This proves the source selection fix, not delivery in the currently pinned Arch package. Updating the package snapshot and exercising the installed artifact remain separate delivery obligations.
