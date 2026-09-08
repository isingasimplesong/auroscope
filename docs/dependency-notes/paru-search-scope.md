# Paru search scope and official selection

Issue #52 includes a later desktop report of AUR-only results and missing colors.
The two screenshots show native repository entries versus an AUR-only wrapper menu.
Successful AUR selection in that report does not exercise an official selection.

## Grounded cause

At pinned Paru `9ac3578807a87858651e81a02586ceb947686e7c`:

- `src/command_line.rs:246-248` maps `-a` to `Mode::AUR`.
- `src/search.rs:457-480` combines the enabled repository and AUR searches.
- `src/lib.rs:342-350` implements interactive machine-target output and status 1
  for sync search independently of that AUR-only flag.

Use `-Ssq --interactive`, not `-Ssaq --interactive`. This removes the wrapper's
unintended AUR-only restriction without replacing Paru's search or configuration.
After an official-only selection, run `-S --` with the resolved targets rather
than passing the original search terms back to Paru. Explicit installs retain
original arguments, native streams and exit status; official packages need no Codex.

## Verification

The exact-argv Go test fails with the old AUR-only flag. The official-selection
regression fails with the old final invocation; both pass with the corrections.
The official test covers status 0 and 42, inherited input, both output streams,
selected targets rather than original multi-word search terms, and no Codex call.

The pinned disposable-Arch selection gate passes for `hello` and `tree`, each
with Color enabled, disabled and redirected output. The `tree` menu contains
both official and AUR entries, and selecting 1 captures exactly `tree`.
The color fix already merged in #49 is retained unchanged, not reimplemented.
These tests do not establish the installed version or configuration on Mathieu's
workstation. Installed-package and full build/install evidence belongs in the
package README; the selection-only gate executes no package recipe.
