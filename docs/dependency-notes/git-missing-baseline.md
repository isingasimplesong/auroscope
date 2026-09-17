# Git: missing historical recipe commits

## Boundary

Issue #89 covers a persistent SQLite baseline whose Git object disappeared after
clone cleanup or upstream history replacement. Keep its historical identity; do
not clear SQLite, classify the current recipe as unchanged, or bypass the audit.

An absent commit produces a full current-recipe bundle with an explicit warning.
The warning reaches both the provider and the user before the audit. Inspection
also shows the old commit. A successful new audit still needs human approval and
the exact final identity guard. Only successful Paru completion advances SQLite.

## Git contract

Consulted local Git 2.47.3 help and exercised:

```sh
git cat-file '--batch-check=%(objectname) %(objecttype)'
```

Send exactly one validated 40-character lowercase hexadecimal commit on stdin.
Git returns `<oid> commit` for an available commit or `<oid> missing` for an
absent object, with exit status zero. Do not infer absence from a generic exit
failure or localized human diagnostics. Reject nonempty stderr, unexpected output
and non-commit objects. In particular, corrupted objects can generate diagnostics
alongside a missing result; those are not a reason to silently lose the diff.

The actual diff remains authoritative when the commit exists. A missing tree or
another diff error still fails preparation rather than becoming a full audit.

## Verification

The clone-cleanup and removed-upstream-history cases reproduced the original
`bad object` failure before the change. Source tests now cover those cases, full
context, preserved history, existing diffs, unchanged recipes, corrupted commits
and missing historical trees.

`TestMissingBaselineReview` exercises the real executable with inert Paru/Codex
fixtures: approval, inspection, cancellation, EOF, skip, invalid audit, final
Paru failure and post-approval identity drift. It also accepts an installed binary
through `AUROSCOPE_TEST_BINARY` for the Arch release gate.

The supported-Paru gate additionally removes its own edited fixture clone while
retaining SQLite, then requires a new audit and approval before a real rebuild.
That package gate must run after the immutable source checkpoint is published;
source-test success alone is not package delivery or live-provider qualification.
