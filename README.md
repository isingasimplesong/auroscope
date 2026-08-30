# AURoscope

AURoscope is a terminal-first safety review wrapper around [Paru](https://github.com/Morganamilo/paru) and Pacman.

It preserves Paru's normal command-line and interactive package-selection experience, while inspecting AUR recipe changes before they are built. Deterministic checks and constrained LLM analysis provide evidence; the user always makes the final decision.

## Intended interface

```console
auroscope                    # normal system upgrade, like bare paru
auroscope <search terms>     # Paru's interactive numbered selection
auroscope -S <packages>      # explicit installation
auroscope status             # human-readable current filter state
auroscope held               # recipes currently awaiting a decision
auroscope explain <package>  # findings and decision history
```

AURoscope is currently a specification-first, from-scratch project. No production implementation exists yet.

## Design principles

- Paru remains the selector, dependency resolver, AUR builder, and Pacman frontend.
- Pacman remains authoritative for official repository transactions.
- Deterministic findings and LLM assessments are advisory evidence, never autonomous decisions.
- Interactive users retain final authority to approve, inspect, defer, or reject an exact recipe.
- Approvals bind to immutable Git commits and file hashes.
- A minimal Paru `PreBuildCommand` prevents changes between review and build.
- SQLite is the source of truth; readable status views are generated from it.
- Reports use ordinary text, Markdown, unified diffs, `$VISUAL`, and `$EDITOR` without editor-specific plugins.

See [the product specification](docs/specification.md) for the agreed behavior.

## Previous project

AURoscope starts from scratch rather than evolving the earlier hook-centered implementation. The previous project remains available as a technical reference:

- [2027a/paru-llm-audit](https://git.2027a.net/2027a/paru-llm-audit)

Its scanner ideas, fixtures, and lessons may be consulted deliberately, but its architecture and code are not inherited by default.
