# AURoscope — High-level product specification

## 1. Purpose

AURoscope becomes the user's usual Arch Linux package-management entry point. It wraps Paru and Pacman while adding a human-controlled review step before AUR recipes are built.

```text
User
  └─ AURoscope
       ├─ selection and dependency resolution: Paru
       ├─ official packages: Pacman, normally through Paru
       ├─ AUR inspection: deterministic rules + constrained LLM
       ├─ decision: user
       └─ build and installation: Paru/Pacman
```

AURoscope does not replace Paru's TUI, dependency resolver, build machinery, or Pacman.

## 2. Paru-compatible interface

The primary forms are:

```console
auroscope
auroscope <search terms>
auroscope -S <packages>
auroscope -Syu
auroscope <other Paru arguments>
```

- Bare `auroscope` behaves like bare `paru`, including the normal system-upgrade flow.
- Search and numbered package selection are delegated to Paru's native `--interactive` interface; AURoscope consumes the selected package names.
- Queries and operations that require no AUR review are passed through with terminal behavior, output, and exit status preserved as far as practical.
- Any operation that would build or install an AUR recipe enters the inspection workflow.

## 3. Transaction planning

Before changing the system, AURoscope identifies:

- explicit targets and relevant dependencies;
- each package's authoritative origin;
- affected AUR package bases (`pkgbase`);
- inspections that can be reused and candidates requiring review;
- dependency consequences of deferring or rejecting a recipe.

Planning and execution are separate Paru resolver runs. After review and approval, Paru resolves the transaction again; AURoscope compares the resulting versions, origins, dependencies, and recipe identities with the approved plan. Any drift stops execution and returns the transaction to review. Only versioned, contract-tested Paru output may be parsed; human-facing output is excluded except for ADR-0002's isolated, temporary, fail-closed Paru 2.1.0 selection adapter. Paru remains the resolver. These boundaries are recorded in [`ADR-0002`](decisions/0002-paru-native-selection-compatibility.md) and [`ADR-0003`](decisions/0003-multi-stage-paru-orchestration.md).

Official Arch repository upgrades must remain supported, complete transactions. AUR packages may be deferred, together with dependants that cannot safely proceed; the reason must be shown.

AURoscope v1 inspects AUR recipes only. It rejects `-B`, targetless `-U`, local/path-like targets, modes containing `pkgbuilds`, and unexpected PKGBUILD-repository plan records before starting Paru. Supported intercepted flows use final trusted repo/AUR mode-reset flags so a configured PKGBUILD repository cannot enter the transaction implicitly. Explicit `-U` package archives remain a pass-through outside recipe inspection. This boundary is recorded in [`ADR-0014`](decisions/0014-reject-local-pkgbuild-inputs-in-v1.md).

## 4. Recipe identity and acquisition

For each AUR `pkgbase`, AURoscope collects recipe data without sourcing the `PKGBUILD`:

- candidate Git commit OID;
- `PKGBUILD`, `.SRCINFO`, `.install` files, patches, units, rules, and other tracked files;
- relevant maintainer and source metadata;
- hashes of all inspected files.

The candidate commit is an immutable identity, not a risk signal. It is used to define the diff, cache inspections, bind human approval, and detect time-of-check/time-of-use substitution at the pre-execution guard. File hashes additionally detect local or workspace modifications. Mutation after the guard by a separate process running as the same Unix user remains outside this guarantee.

A first installation receives a full recipe review. Later reviews primarily compare the last approved/built identity with the candidate, while including enough complete file context to interpret the diff.

## 5. Deterministic findings

Local rules emit immutable, evidence-backed findings with at least:

```json
{
  "source": "deterministic",
  "rule": "download-piped-to-shell",
  "severity": "critical",
  "file": "PKGBUILD",
  "line": 47,
  "evidence": "curl -fsSL \"$url\" | bash"
}
```

Rules may detect direct execution of downloads, obfuscation, sensitive home-directory access, new domains, `SKIP` checksums, install scripts, systemd/udev/polkit/sudoers changes, dangerous permissions, setuid/capabilities, and changes to `provides`, `conflicts`, or `replaces`.

A finding is an observation, never a decision. Its evidence and rule-defined severity cannot be silently rewritten, removed, upgraded, or downgraded by the LLM or an aggregate score.

## 6. Constrained LLM assessment

The model receives the diff, necessary changed-file context, selected metadata, and deterministic findings explicitly labelled as untrusted package content and immutable scanner output.

It may explain changes, identify contextual relationships, highlight uncertainty, and suggest points for human attention. It may not decide installation, modify deterministic findings, invoke tools, access the host/network/secrets, or present its output as proof. Its response must pass a strict JSON schema.

## 7. State model and human authority

Three separate axes prevent signals from masquerading as decisions.

### Technical inspection status

- `complete`
- `partial`
- `failed`

### Advisory signal level

- `clear`
- `informational`
- `caution`
- `high`
- `unknown`

### Human decision

- `approve`
- `inspect`
- `defer`
- `reject`

Before the user acts, `decision` is null. In the default interactive `human-authority` policy:

- clear recipes remain in the normal transaction plan and its ordinary confirmation;
- any noteworthy signal, partial analysis, or error is paused and presented to the user;
- the user may inspect, approve the exact recipe, defer it, reject it, or cancel the transaction;
- neither a high signal nor a scanner/model failure is an autonomous veto;
- neither failure nor uncertainty becomes silent approval.

Non-interactive modes may later apply an explicit configured policy, but cannot change the default interactive semantics.

An explicit approval is transaction-scoped and one-shot. It binds the source/namespace/`pkgbase`, commit and tree, complete canonical recipe manifest and per-file identity, inspection and human decision, workspace, expected Paru process, and expiry. `PKGBUILD` is included but is not sufficient by itself. The default approval lifetime is 30 minutes, with a configurable maximum of 2 hours. The exact binding, atomic claim, replay prevention, and invalidation rules are recorded in [`ADR-0010`](decisions/0010-one-shot-approval-protocol.md).

## 8. Generic terminal review

Reports use ordinary text, Markdown, unified diffs, and normal files. The review command resolves the viewer in this order:

1. `$VISUAL`;
2. `$EDITOR`;
3. a terminal pager fallback.

After the viewer closes, a plain terminal menu offers approval, further inspection, deferral, rejection, or cancellation. No Vim, Neovim, Emacs, or other editor plugin is required.

If the user modifies recipe content, the prior identity and approval are invalidated. Modified content must be rehashed and reinspected before it can be built.

## 9. Execution and TOCTOU guard

After human decisions, AURoscope delegates the executable plan to Paru/Pacman. Deferred or rejected AUR recipes and impossible dependants are omitted with an explanation; official repository upgrades are not fragmented into unsupported partial upgrades.

Paru's `PreBuildCommand` remains a narrow guard only. After AURoscope review and before any recipe-supplied code is executed, it performs the last complete identity check of every planned recipe against a live AURoscope approval. The check covers the reviewed commit, tree, tracked-file manifest, and file hashes; any divergence aborts the operation and returns the recipe to review.

Paru does not invoke this hook immediately before each individual build: it may perform other trusted orchestration, including official dependency installation, between the hooks and `makepkg`. AURoscope therefore runs the execution phase with `--skipreview` so Paru cannot edit recipe content after the guard. A separate same-UID process can still race the worktree after the guard returns; eliminating that residual race requires a stronger isolation boundary and is outside the current threat model. This accepted boundary is recorded in [`ADR-0005`](decisions/0005-recipe-identity-guard-boundary.md).

The handoff carries only a non-secret transaction ID in a private transaction-specific Paru configuration. The guard verifies the recorded Paru process and workspace, computes the complete identity, atomically claims one matching unexpired SQLite approval, and computes the identity again before returning success. Approval progresses only `armed → claimed → consumed`; mismatch, expiry, cancellation, failure, signal, or another terminal transaction state invalidates all reusable approval state. A claimed or consumed approval cannot authorize another process or transaction. This protocol is recorded in [`ADR-0010`](decisions/0010-one-shot-approval-protocol.md).

## 10. SQLite and readable filter state

SQLite is the sole authoritative store for:

- package bases and recipe identities;
- candidate and previous commits;
- inspected-file hashes;
- deterministic findings and LLM assessments;
- technical status and advisory signal;
- human decisions and their exact scope;
- build/install outcomes;
- report, model, and prompt-version metadata.

Human-readable state is generated from SQLite rather than maintained as a second mutable state file:

```console
auroscope status
auroscope status --verbose
auroscope held
auroscope explain <package>
auroscope status --json
auroscope status --markdown
```

The default status view reports at least:

- total known AUR package bases;
- number currently clear, awaiting review, deferred, or rejected;
- names of held recipes;
- concise reasons and signal sources;
- dependency consequences;
- the human action currently required.

`--markdown` may export a readable snapshot for sharing or archiving, but that export is never authoritative and must not be read back as state.

## 11. Technology, packaging, and storage constraints

- AURoscope is implemented in Go and delivered as a small target-specific executable, initially for Arch Linux on `linux/amd64`.
- Prefer the Go standard library. Direct dependencies must be ordinary, maintained, minimal, and justified by a concrete need; dependency minimization must not cause bespoke reimplementation of fundamental components.
- Beyond the SQLite driver, the accepted initial direct dependencies are `github.com/pelletier/go-toml/v2` for strict TOML configuration and `github.com/mattn/go-shellwords v1.0.14` only for non-expanding `$VISUAL`/`$EDITOR` argv parsing. Migrations, LLM validation/client code, and logging use the standard library; no PTY or framework dependency is added without demonstrated need.
- The initial Arch distribution is a self-hosted AUR-style PKGBUILD repository. Publishing to `aur.archlinux.org` is deferred. Go should be a build dependency rather than a runtime dependency where feasible.
- SQLite remains the authoritative durable state store. The Arch `linux/amd64` v1 uses pinned `github.com/mattn/go-sqlite3` with CGO; schema, migrations, concurrency, and recovery remain separate design-phase decisions.
- The accepted concrete storage layout is:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/auroscope/config.toml                  0600
${XDG_STATE_HOME:-$HOME/.local/state}/auroscope/state.db                0600
${XDG_STATE_HOME:-$HOME/.local/state}/auroscope/reports/YYYY/MM/...     0600 files, 0700 dirs
${XDG_STATE_HOME:-$HOME/.local/state}/auroscope/backups/                 0700
${XDG_CACHE_HOME:-$HOME/.cache}/auroscope/recipes/<source>/<pkgbase>/    0700 root
${XDG_CACHE_HOME:-$HOME/.cache}/auroscope/model/                         0700, optional
${XDG_RUNTIME_DIR}/auroscope/<transaction>/                              preferred, 0700
${TMPDIR:-/tmp}/auroscope-<uid>-<random>/<transaction>/                  fallback, 0700
```

- AURoscope uses umask `0077`, private roots, and `0600` sensitive files including SQLite `-wal`/`-shm`. It rejects relative roots, unsafe ownership or group/world-writable roots, and symlinks at sensitive final components. If a safe `XDG_RUNTIME_DIR` is unavailable, it uses an unpredictable private `os.MkdirTemp` fallback under `$TMPDIR`; `/tmp` is not assumed to be RAM.
- Temporary clones and downloaded audit inputs are removed after success, error, interruption, or cancellation. Descriptor-relative no-follow recovery cross-checks recorded UID, device/inode, and PID plus process start time before removing stale direct children; active work is never removed, and stale runtime handoffs have a 24-hour grace period.
- Default retention is finite and configurable: 365 days for recipe identities, human decisions, and build/install outcomes; 30 days for reports and retained model JSON; 7 days for SQLite backups; 14 days or 512 MiB LRU for reconstructible recipe cache, whichever limit is reached first; and 3 days for failed-work metadata. No category defaults to unlimited retention or disabled purge.
- Human-readable status is generated from SQLite. Markdown and JSON exports are snapshots, never a second state source.

Accepted design decisions relevant to these constraints are recorded in:

- [`ADR-0001`](decisions/0001-go-and-self-hosted-arch-packaging.md);
- [`ADR-0002`](decisions/0002-paru-native-selection-compatibility.md);
- [`ADR-0007`](decisions/0007-mattn-go-sqlite3-cgo.md);
- [`ADR-0008`](decisions/0008-minimal-direct-go-dependencies.md);
- [`ADR-0010`](decisions/0010-one-shot-approval-protocol.md);
- [`ADR-0012`](decisions/0012-xdg-layout-permissions-retention.md);
- [`ADR-0014`](decisions/0014-reject-local-pkgbuild-inputs-in-v1.md).

## 12. Design before implementation

Before production code, the design phase must investigate and submit proposals for:

- exact Paru/Pacman command, PTY, origin, dependency, transaction, cache, and exit-code contracts;
- Go module boundaries, CLI/process handling, dependencies, build, and release strategy;
- SQLite schema, migrations, lifecycle, concurrency, retention, and crash recovery;
- approval identity, expiry, anti-replay, and `PreBuildCommand` TOCTOU protocol;
- deterministic rule and constrained-LLM schemas, privacy, truncation, and failure behavior;
- concrete XDG paths, permissions, configuration format, reports, and cleanup;
- threat model and risk-proportional test strategy.

Consequential alternatives must be presented to Mathieu and recorded as accepted ADRs. The detailed implementation plan is written only after those decisions are accepted. See [`docs/design-phase.md`](design-phase.md).

## 13. Initial scope

Included in the first useful version:

- daily replacement for Paru;
- bare upgrade, explicit installation, and Paru-native interactive selection;
- differential AUR recipe inspection;
- deterministic rules and constrained LLM assessment;
- exclusively human decisions in interactive mode;
- generic editor/pager reports;
- SQLite persistence and readable status commands;
- commit/hash-bound approval and Paru pre-build guard;
- structured JSON output and stable exit categories.

Deferred:

- a replacement TUI;
- editor-specific plugins;
- exhaustive upstream-source or compiled-binary analysis;
- a custom build sandbox;
- a generic plugin/rule framework;
- replacement of Paru's resolver;
- local PKGBUILD builds and configured PKGBUILD repositories.

## 14. Acceptance criteria

AURoscope is not useful until tests demonstrate that:

1. bare `auroscope` can replace bare `paru` in the normal upgrade workflow;
2. Paru's numbered interactive selection is preserved;
3. official packages proceed without AUR-specific friction;
4. AUR recipes and required dependencies are inspected before build;
5. deterministic and LLM evidence remain visibly separate;
6. all consequential interactive decisions belong to the user;
7. held recipes and dependency effects are readable through `status`, `held`, and `explain`;
8. approval applies only to the exact inspected commit and hashes;
9. a recipe changed after approval and before or during the guard is stopped before recipe-supplied code executes;
10. cancellation leaves no reusable floating approval;
11. end-to-end fixtures run in a disposable Arch environment without altering the real workstation.

## 15. Relationship to the previous project

AURoscope is a from-scratch successor to [2027a/paru-llm-audit](https://git.2027a.net/2027a/paru-llm-audit). The old project is a reference for lessons, test fixtures, and scanner ideas only. No production code or hook-centered architecture is inherited implicitly.
