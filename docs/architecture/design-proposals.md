# AURoscope technical design proposals

## Status and decision protocol

**Status:** Proposed for Mathieu's review except where an accepted ADR is linked explicitly. A recommendation alone is never acceptance.

This document intentionally contains no implementation plan and no production code. As Mathieu accepts, rejects, or amends the numbered decisions in [Decision set](#decision-set-for-mathieu), accepted choices are split into ADRs. Only after the consequential ADRs are accepted may `docs/implementation/initial-plan.md` be written.

## 1. Grounded baseline and proof

### Versions investigated

- Paru stable upstream: `v2.1.0`, commit `70f66dc9eddb40e264ee6c9197541262b7792c9c`.
- Paru AUR recipe: `2.1.0-2`, AUR commit `329be2113c590046cb29858c23d9b96a8d7bd586`.
- Paru post-release reference: upstream `master` commit `9ac3578807a87858651e81a02586ceb947686e7c`; selection-stream fix commit `d1dfbc48895a2a3b3fadc14569c3f2ffb5112b2e`.
- Pacman/makepkg upstream: `v7.1.0`, commit `5683f8477a0afcc6b331766175a83445b2dcfe89`.
- Current Arch Pacman package: `7.1.0.r9.g54d9411-2`, packaging commit `abdc0dfedf3ca553a02dde8551e972fe745535b7`, upstream patch-level commit `54d94116164b0b2202c6061c4a59c6f3e70820d8`.[9]
- Go spikes: host Go `1.24.4`; current Arch package observed as `2:1.27.0-1` in a disposable container.

Paru's manuals state that bare invocation is `paru -Syu`, bare search terms are an alias for sync interactive selection, `--interactive` is intended to feed selected packages to another tool, and `PreBuildCommand` runs once per package base via `sh -c` in the recipe directory with `PKGBASE` and `VERSION` set.[1][2]

Pacman 7.1 documents repository transaction printing, complete `-Syu` semantics, query/origin filters, cache behavior, and the distinction among `--root`, `--sysroot`, `--dbpath`, and absolute cache/hook paths.[10]

Makepkg documents its output roots and exit codes, but its implementation sources the PKGBUILD before handling `--printsrcinfo`, `--packagelist`, or `--verifysource` results.[11][12][13]

### Disposable spikes performed

1. **Makepkg execution boundary.** In `archlinux:base-devel` with makepkg `7.1.0`, a PKGBUILD with a harmless top-level marker write was invoked separately with `--printsrcinfo`, `--packagelist`, and `--verifysource`. Every command exited `0` and wrote the marker. Therefore none is a safe inspection parser.
2. **Paru/libalpm compatibility.** Building the exact `v2.1.0` lockfile against libalpm `16.0.1` failed because `alpm 4.0.3` supports libalpm 15. Repeating the AUR recipe's `cargo update alpm alpm-utils` produced a working `paru v2.1.0 - libalpm v16.0.1`.[8]
3. **Stable Paru selection stream.** With the current AUR-style `v2.1.0` build, `paru -Ssaq --interactive paru` put the menu and selected name on stdout, stderr was empty, and the process exited `1`. The cause is the released file-descriptor code.[3][5]
4. **Post-release selection stream.** Upstream commit `d1dfbc4` fixes stdout redirection.[6] At `9ac3578`, the same spike put only `paru` on stdout, the menu on stderr, and still exited `1`.[7]
5. **SQLite driver comparison.** Minimal foreign-key/WAL programs were built and run. `github.com/mattn/go-sqlite3 v1.14.50`: 3,608,224-byte stripped binary, 2 modules total, CGO required at runtime. `modernc.org/sqlite v1.57.0`: 6,299,940-byte stripped binary, 26 modules total, pure-Go cross-build succeeded, and this version requires Go 1.25 or newer.
6. **Paru config and local-build boundary.** A generated config that included a user config containing `PreBuildCommand = /bin/false` and then set AURoscope's command successfully ran the later command, confirming simple override order. The same benign `paru -B` spike printed `Generating .SRCINFO` and invoked makepkg before `PreBuildCommand`; local PKGBUILD planning therefore crosses the execution boundary before the guard. A second config defined a hostile-marker PKGBUILD repository with forced SRCINFO generation; `paru -P --order --mode=ar -- paru` left the marker absent, proving the forced mode clears configured PKGBUILD repositories before scanning on the inspected post-fix commit.

Detailed evidence is preserved in `docs/dependency-notes/`.

## 2. Exact Paru/Pacman contract

### Options

#### Option A — transparent wrapper plus pre-build audit only

Run Paru once with inherited terminal streams; use `PreBuildCommand` to audit and prompt.

**Advantages**

- closest to bare Paru;
- minimal resolver duplication;
- works with stable Paru 2.1 selection.

**Costs and rejection reason**

- `PreBuildCommand` occurs after planning and download, before Paru's own review, not immediately before each actual build.[4]
- it cannot prepare a complete review before execution, and cannot safely separate deterministic evidence, model evidence, and durable human decisions at the wrapper level;
- Paru's own review can mutate files after the guard;
- a same-version cached artifact can be installed although approval covered only recipe files.

This does not satisfy the accepted specification without changing the meaning of the pre-build hook.

#### Option B — two-stage Paru orchestration with a narrow guard

Use Paru's supported machine-oriented surfaces for selection/planning, inspect and decide in AURoscope, then invoke Paru for execution with a transaction-scoped `PreBuildCommand`.

**Advantages**

- preserves Paru selection and resolver instead of copying them;
- review and approval happen before executable Paru work;
- `PreBuildCommand` remains a narrow identity check;
- clear place to persist the plan, evidence, decisions, and outcomes.

**Costs**

- planning and execution are separate resolver runs, so the guard must reject drift;
- current stable Paru 2.1 cannot cleanly separate menu output from selected targets;
- no single documented Paru JSON transaction-plan API exists.

**Recommendation:** Option B, with explicit compatibility gates and no durable parsing of human UI beyond ADR-0002's strictly bounded transitional 2.1.0 adapter.

**Decision:** Option B was accepted on 2026-08-30. Paru remains responsible for selection, resolution, build, and installation across separate planning and execution runs; any drift from the reviewed plan returns to review. See [`ADR-0003`](../decisions/0003-multi-stage-paru-orchestration.md).

#### Option C — link to or reimplement Paru/libalpm internals

**Advantage:** richer typed plan.

**Costs and rejection reason:** makes AURoscope a second Paru frontend/resolver, couples Go to Rust/libalpm internals, and violates the product boundary.

### Argument classification

| Input class | v1 behavior | Reason |
|---|---|---|
| `--help`, `--version`, completion, pure `-Q`, `-F`, `-D`, `-T` | transparent pass-through | no AUR recipe build |
| pure `-R`, `-Sc`, `-G`, `-L`, `-C`, non-installing `-P` | transparent pass-through, preserving terminal/status | may touch Paru state but does not build a recipe |
| `-Ss`, `-Si`, `-Sl`, `-Qu` without install | pass-through | informational only |
| bare search terms | intercept selection → plan → review → execute | may select AUR recipes |
| `-S <targets>` | intercept unless an explicit trusted repo-only mode proves no AUR build is possible | explicit install may resolve AUR targets/dependencies |
| bare invocation, `-Su`, `-Syu`, `-Sua` | intercept upgrade flow | may build installed foreign packages |
| `-B <dirs>`, bare targetless `-U`, path-like sync targets (including `./...`), and targets resolved from PKGBUILD repositories | reject as unsupported in v1 before starting Paru | local paths can invoke makepkg `--printsrcinfo` before `PreBuildCommand`; safe support needs a separate design |
| `-U <package files/URLs>` | pass-through to Paru/Pacman in v1 | installs package archives, not AUR recipes; package-archive inspection is deferred |
| `--downloadonly` with AUR target | preserve Paru rejection | Paru 2.1 explicitly rejects this combination |
| `--noconfirm` with any new AUR decision | reject with AURoscope policy exit | conflicts with default human authority; later automation needs an explicit policy ADR |
| ambiguous/unknown operation combination | pass-through only if classification proves it cannot build; otherwise fail before mutation | fail safe at the boundary, not by guessing |

Argument parsing must classify only AURoscope-owned subcommands and the minimum Pacman/Paru operation grammar. It must retain the original argv byte-for-byte for transparent execution; it must not normalize unknown options.

### Native interactive selection

Proposed command for search selection:

```text
paru -Ssaq --interactive -- <terms>
```

The menu inherits the real terminal through stderr; stdout is captured as newline-delimited selected package names. The observed success contract is unusual: Paru's sync-search path returns `1`, including when selection succeeds.[3] AURoscope therefore accepts this `1` only when:

- the subprocess is exactly the selection stage;
- stdout contains one or more syntactically valid selected names;
- every line is bounded and contains no control characters;
- the child was not terminated by a signal.

Cancellation is `1` plus empty selection and maps to AURoscope `cancelled`, not `failed`.

**Accepted compatibility boundary:** the durable target is the first stable Paru release containing `d1dfbc4`, but Paru 2.1.0 may be supported meanwhile by an isolated, explicitly temporary adapter for its verified combined stdout format.[5][6] That adapter must accept only bounded, unambiguous tested output and fail closed on format or locale divergence. A pinned `paru-git`/upstream commit remains limited to design, development, and contract tests. Both adapters require executable capability tests; the version string alone is insufficient. See [ADR-0002](../decisions/0002-paru-native-selection-compatibility.md).

### Planning and authoritative origin

1. Paru remains authoritative for AUR/repository origin and dependency classification within supported modes. The classifier interprets user mode selectors but never forwards them blindly: any mode containing `pkgbuilds`/`p` is rejected; repo-only is normalized to a final trusted `--repo`, AUR-only (including `-Sua`) to a final trusted `--aur`, and default/combined repo+AUR to trusted `--repo --mode=aur`. These flags are inserted after user options and before the target separator, so they reset config-provided `PkgbuildsOnly` rather than OR-ing with it. Explicit local build operations and local path targets (`./`, `../`, absolute paths, or `file:`; repository-qualified `repo/pkg` remains valid) fail before any Paru process starts. Any `PKGBUILD` record is a contract violation and aborts.
2. For explicit selected targets, use Paru's `paru -P --order -- <targets>` output as a version-gated planning surface. Released `v2.1.0` documents repository records as `INSTALL TARGET|DEP|MAKE <repo> <name>`, while post-release commit `9ac3578` actually emits `REPO TARGET|DEP|MAKE <repo> <name>` from `src/order.rs` although its man page still says `INSTALL`; AUR records remain `AUR TARGET|DEP|MAKE <pkgbase> <names...>`.[1][26]
3. Parse only the grammar proven for the exact supported build. Reject unknown record kinds, documentation/code mismatches, `MISSING`, and conflicts rather than treating them as harmless. The `--order` spike returned `0` with complete synced databases and `1` plus `MISSING` records when repository databases were absent.
4. Confirm repository packages independently with Pacman `-S --print --print-format` where a repository transaction is about to run. Pacman's repository order is authoritative.[10]
5. Committed `.SRCINFO` and AUR RPC metadata are planning data only. Never regenerate `.SRCINFO` with makepkg during inspection.
6. Before execution, clone/fetch each planned `pkgbase`, resolve the candidate commit, enumerate the Git tree, and inspect that immutable candidate.
7. During execution, an unplanned AUR `PKGBASE` reaching the guard aborts the whole AUR phase before repository build dependencies are installed; Paru 2.1 runs all pre-build commands before its repository dependency installation.[4]

**Unresolved proof:** bare sysupgrade has no documented single machine-readable `--order` equivalent for the full combined repository+AUR plan. The recommended upgrade split below avoids claiming one.

### Complete official upgrades and deferred AUR work

#### Option A — Paru combined upgrade with injected ignores

Keeps one combined flow, but refresh may occur before AUR review, failure after refresh leaves the user responsible for completing the upgrade, and deriving a stable complete plan is weak.[1][2]

#### Option B — explicit repository phase, then AUR phase

1. Before any package transaction, discover AUR updates with the exact supported Paru query adapter (for the inspected branch, quiet AUR upgrades are emitted one package name per line by `paru -Quaq`), force supported `aur,repo` mode, feed AUR names through the version-gated `--order` planner, clone, inspect, and collect human decisions. This preliminary plan uses the current repository databases and explicitly labels dependency consequences provisional.
2. Run the complete official repository upgrade through Paru's repository-only mode (`paru -Syu --repo`) with inherited terminal and exact child status.
3. Re-query and re-plan the AUR phase against the now-current repository state. Any new candidate, changed identity, origin/provider change, or dependency-closure change returns to human review before AUR execution.
4. Compute the exclusion closure from the final proven plan (every package emitted by a held `pkgbase`, plus dependants that cannot resolve without it), then execute `paru -Sua` with one exact `--ignore` value for each excluded package; reject any unplanned guard call. If Paru's fresh resolution reports a different closure, stop and return to review rather than installing a guessed subset.

**Recommendation:** Option B. It preserves the supported complete repository transaction and makes AUR deferral honest. The trade-off is two visible phases and the possibility that an AUR package is temporarily incompatible after the official upgrade. That residual risk must be shown before phase 1 when known, but must not be “fixed” by withholding official repository packages and creating a partial upgrade.

### Execution hardening and cache behavior

- Generate a transaction-specific Paru config in a private runtime directory and select it with `PARU_CONF`; do not edit the user's Paru config. Before overriding the environment, resolve the original effective config (`PARU_CONF`, XDG user file, then `/etc/paru.conf`). If it exists, the generated file first `Include`s that exact trusted absolute path and then declares a final `[bin]` section with AURoscope's `PreBuildCommand`, so the transaction guard wins by parse order. Paru's parser reads `Include` immediately and later directives replace the optional command value.[25] Contract-test nested includes, missing files, repeated sections, and paths containing whitespace.
- Force `--skipreview` during the execution run because AURoscope already reviewed the recipe and Paru's own file-manager review occurs after the pre-build hook.[2][4]
- Never add `--noconfirm` to the user's final Pacman confirmation.
- A recipe approval does not approve an arbitrary cached package file. Default v1 rule: force rebuild unless SQLite contains a successful AURoscope build record for the same recipe identity and exact package-artifact SHA-256. Reusable artifacts are verified before `pacman -U`.
- Do not run `makepkg --packagelist` to predict output during inspection; it executes top-level PKGBUILD code.[11][12]
- Preserve child exit status for transparent flows. For intercepted flows store: normalized AURoscope category, exact child exit code, and terminating signal separately.
- On SIGINT/SIGTERM/SIGHUP, forward to the child's process group, wait, invalidate unconsumed approvals, clean runtime handoff, and exit as signal-derived status. Go's `os/exec` does not invoke a shell implicitly; process-group and signal policy must be explicit.[18][19]

### Minimum compatibility boundary

- Pacman/makepkg: support `7.1.x` initially; test both upstream `v7.1.0` and Arch's packaged patch level.
- Paru execution features: `2.1.0-2` behavior is the inspected floor for planning/guard mechanics.
- Paru native selection capture: the durable floor is the first stable release containing `d1dfbc4`; until it exists and passes the capability matrix, allow a strictly bounded temporary adapter for verified Paru 2.1.0 output.
- At startup, verify `paru --version`, `pacman --version`, and `makepkg --version`; run feature-level contract tests in release CI. Version strings alone cannot distinguish patched Paru 2.1 builds, and the temporary adapter must reject output outside its tested grammar.

## 3. Go architecture and direct dependencies

### Proposed package boundaries

```text
cmd/auroscope          command entry only
internal/cli           classify argv and AURoscope subcommands
internal/config        XDG discovery, defaults, validation
internal/process       argv-only subprocesses, process groups, signals
internal/paru          versioned Paru adapter and parsers
internal/recipe        Git identity, safe manifest, diff/context
internal/scanner       deterministic immutable findings
internal/llm           inference-only HTTP contract and validation
internal/review        terminal presentation and human decisions
internal/store         SQLite transactions, migrations, queries
internal/approval      approval lifecycle and guard command
internal/report        text/Markdown/JSON rendering from typed state
```

This is one executable and one process except for deliberate external commands. Packages are domain boundaries, not interfaces-for-everything. Define interfaces only at actual nondeterministic boundaries used by tests: clock/randomness, process runner, model transport, and store transaction.

### CLI/process options

- **Framework CLI:** convenient help generation, but risks fighting passthrough semantics. Not recommended.
- **Standard library parser/classifier:** retain raw argv and parse only enough grammar to identify owned subcommands and mutation risk. Recommended.
- **PTY for all Paru calls:** maximizes TTY likeness but multiplexes output and makes machine capture fragile. Rejected.
- **Direct inherited descriptors:** pass-through uses `Stdin/Stdout/Stderr = os.*`; selection inherits stdin+stderr and captures only stdout. Recommended.
- **`github.com/creack/pty`:** defer. Add only if an executable contract test proves a required Paru/Pacman prompt refuses inherited descriptors without a controlling terminal.[22]

### SQLite driver options

#### `github.com/mattn/go-sqlite3`

- small dependency surface and smaller measured binary;
- embeds SQLite through CGO; Arch `base-devel` makes the build practical;
- target-specific CGO builds complicate cross-compilation, but v1 targets only Arch `linux/amd64`.[20]

#### `modernc.org/sqlite`

- pure Go and easy target cross-build;
- larger measured binary and dependency graph;
- current tested version requires Go 1.25+, which current Arch satisfies but the present host Go 1.24.4 does not without toolchain download.[21]

**Accepted in [ADR-0007](../decisions/0007-mattn-go-sqlite3-cgo.md):** `mattn/go-sqlite3 v1.14.50` with CGO for v1. It fits the single target, is materially smaller in the spike, and avoids 24 extra modules. Reconsider a pure-Go driver when cross-target releases become a real requirement. Pin the exact module and SQLite compile options; verify `PRAGMA foreign_keys`, WAL, busy timeout, migrations, integrity/corruption handling, backup/restore, and concurrency in tests.

### Other proposed dependencies

| Need | Proposal | Justification |
|---|---|---|
| config | `github.com/pelletier/go-toml/v2` pinned at design implementation time | readable TOML without inventing a parser; strict unknown-field validation required.[23] |
| SQLite | `github.com/mattn/go-sqlite3` | fundamental database driver; CGO accepted for Arch v1.[20] |
| PTY | none initially | inherited descriptors satisfy the post-fix Paru selection contract |
| migrations | none | ordered SQL files embedded with `//go:embed`, applied transactionally |
| JSON Schema runtime | none initially | typed `encoding/json`, `DisallowUnknownFields`, size bounds, enum/range/cross-field validation |
| LLM SDK | none | a narrow OpenAI-compatible HTTP adapter is smaller and easier to constrain |
| logging framework | none | structured records in SQLite plus concise stderr diagnostics |
| shell-word parser | `github.com/mattn/go-shellwords v1.0.14` | parse common `$VISUAL`/`$EDITOR` values with arguments without invoking a shell; environment and backtick expansion must remain disabled.[27] |

### Build/release

- Build in a clean Arch environment using the repository PKGBUILD; Go and C toolchain are build dependencies, not runtime dependencies.
- Use `-trimpath`, inject version/commit with `-ldflags`, and emit checksums plus SBOM/module list.
- Do not promise a static or distro-independent binary. The accepted target is Arch `linux/amd64`.
- Rebuild twice in clean containers and compare artifacts; if bit-for-bit reproducibility is not achieved, document the variance before release.

## 4. SQLite state model

### Options

- **Event log only:** excellent audit trail, awkward status queries and invariant enforcement.
- **Mutable current-state tables only:** simple queries, weak history and replay evidence.
- **Normalized immutable evidence/history plus explicit mutable lifecycle rows:** recommended balance.

### Proposed schema

All IDs are opaque 128-bit random values stored as 16-byte BLOBs; timestamps are UTC Unix milliseconds. Enums use `TEXT CHECK (...)`. All foreign keys are enforced.

- `schema_migrations(version INTEGER PRIMARY KEY, applied_at, sha256)`
- `package_bases(id, source_kind, source_namespace, pkgbase, created_at, UNIQUE(source_kind, source_namespace, pkgbase))`
- `recipe_identities(id, package_base_id, commit_oid, tree_oid, manifest_sha256, created_at, UNIQUE(package_base_id, commit_oid, manifest_sha256))`
- `recipe_files(recipe_identity_id, path_bytes, mode, object_type, size, sha256, git_blob_oid, symlink_target_bytes, PRIMARY KEY(recipe_identity_id, path_bytes))`
- `inspections(id, recipe_identity_id, baseline_identity_id, scanner_schema, scanner_version, status, advisory_signal, context_manifest_sha256, started_at, completed_at, error_code)`
- `deterministic_findings(id, inspection_id, rule_id, rule_version, severity, path_bytes, line, byte_start, byte_end, evidence_sha256, evidence_text)`; append-only
- `llm_assessments(id, inspection_id, contract_version, provider, model, prompt_sha256, request_manifest_sha256, response_sha256, status, advisory_level, summary, raw_json, started_at, completed_at)`
- `human_decisions(id, inspection_id, actor_uid, action, note, created_at)`; append-only, action in `approve|inspect|defer|reject`
- `transactions(id, cwd_bytes, wrapper_version, paru_version, pacman_version, state, owner_uid, wrapper_pid, wrapper_start_ticks, created_at, updated_at, terminal_error)`
- `transaction_args(transaction_id, ordinal, arg_bytes, PRIMARY KEY(transaction_id, ordinal))`
- `transaction_recipes(transaction_id, package_base_id, recipe_identity_id, role, disposition, dependency_reason, PRIMARY KEY(transaction_id, package_base_id))`
- `approvals(id, transaction_id, inspection_id, decision_id, recipe_identity_id, workspace_dev, workspace_ino, manifest_sha256, expires_at, state, claimed_at, consumed_at, invalidated_at, UNIQUE(transaction_id, recipe_identity_id))`; `decision_id` must reference an `approve` action for the same inspection
- `process_sessions(transaction_id PRIMARY KEY, paru_pid, paru_start_ticks, paru_exe_dev, paru_exe_ino, started_at, ended_at)`
- `builds(id, transaction_id, recipe_identity_id, approval_id, status, started_at, completed_at, child_exit, child_signal)`
- `package_artifacts(id, build_id, pkgname, version, arch, path_bytes, size, sha256, signature_status, UNIQUE(build_id, sha256))`
- `install_outcomes(id, transaction_id, artifact_id, status, pacman_exit, installed_at)`

`path_bytes`, `arg_bytes`, and `cwd_bytes` must have reversible representations; do not assume hostile Git filenames or Unix argv are UTF-8. Reports render escaped display strings separately. Critical cross-row invariants are enforced both by composite foreign keys/unique indexes where SQLite can express them and by transactional triggers/store checks: approval, decision, inspection, and recipe identity must match; only an `approve` decision can arm approval; lifecycle transitions are monotonic.

### Lifecycle and transactions

```text
transaction: planning → reviewing → ready → executing → succeeded
                                      ↘ cancelled|failed
approval:    armed → claimed → consumed
               ↘ invalidated|expired
inspection:  running → complete|partial|failed
```

- Use WAL for normal local-filesystem operation, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=FULL` for approval state, and one short write transaction at a time.[15][16][17] Run a locking/WAL self-test when initializing the DB; do not place authoritative state on a network filesystem that cannot provide SQLite's required shared-memory/locking semantics.
- AURoscope takes an application lock file under runtime state to prevent two interactive mutating wrappers. SQLite still permits read-only status commands.
- Never hold a SQLite transaction while waiting on Paru, the model, editor, or user.
- Migrations are embedded, checksummed, monotonic, and applied under an exclusive application lock. No down-migration promise in v1; backup before migration and support restore.
- On startup: check `PRAGMA quick_check`, expire stale approvals, mark orphaned running transactions interrupted after validating PID+start time, and clean only work owned by those recorded transactions.
- Retain recipe identities, decisions, build/install outcomes, and compact findings by policy; purge raw LLM payloads/reports earlier. Never purge an identity referenced by an approval or installed artifact record.

### Retention proposal

- durable identity/decision/outcome: unlimited by default until explicit `auroscope purge --history-before`;
- full reports and raw model JSON: 90 days default;
- reconstructible recipe cache: 30 days or 1 GiB, LRU, whichever binds first;
- failed temporary-work metadata: 7 days while actual private temp trees are removed at recovery;
- `VACUUM` only as an explicit maintenance action; routine purge uses incremental vacuum if configured.

## 5. Approval and anti-TOCTOU protocol

**Decision:** The one-shot transaction-ID handoff, complete identity binding, atomic claim, 30-minute default expiry with a configurable 2-hour maximum, and terminal invalidation rules below were accepted on 2026-08-30. See [`ADR-0010`](../decisions/0010-one-shot-approval-protocol.md). The separately accepted final guard timing remains governed by [`ADR-0005`](../decisions/0005-recipe-identity-guard-boundary.md).

### Identity

An approval binds all of:

```text
transaction_id
source_kind + source_namespace + pkgbase
40/64-hex Git commit OID with detected object format
Git tree OID
canonical tracked-file manifest SHA-256
per-file path bytes + object type + mode + size + SHA-256 + Git blob OID
inspection id and scanner contract version
human decision id
workspace device + inode
expiry
```

The commit OID is identity, not a safety score. The manifest additionally detects local changes, untracked relevant files, worktree substitution, and hash-algorithm migration.

### Handoff options

- **Bearer token in environment:** simple, but inherited by subprocesses and unnecessary before recipe execution. Rejected.
- **One-shot token file:** limits accidental reuse but adds secret-file lifecycle without defeating the same-UID threat.
- **Transaction ID plus process/workspace binding in SQLite:** accepted. The transaction ID is not treated as a secret.

### Accepted protocol

1. Wrapper creates transaction and private runtime directory (`0700`), records its own PID/start time.
2. It plans, clones, safely collects, scans, obtains optional LLM assessment, and records the user's decision.
3. For each approved identity it inserts an `armed` approval with a 30-minute default expiry and a configurable maximum of 2 hours.
4. It writes a transaction-specific Paru config (`0600`) whose `PreBuildCommand` is a fixed shell command equivalent to:

   ```text
   exec /usr/bin/auroscope guard --transaction <id>
   ```

   No package-derived text enters that command. Paru still invokes `sh -c`, but the string consists only of fixed syntax and a validated lowercase-hex transaction ID; the leading POSIX `exec` replaces the shell with the guard.
5. Wrapper starts Paru as a child process group with `PARU_CONF` pointing to that private config, records Paru PID, `/proc/<pid>/stat` start ticks, and executable identity.
6. Guard obtains `PKGBASE` from Paru, treats `VERSION` as advisory, opens cwd without following symlink components, verifies cwd device/inode, repository origin, `HEAD`, tree, tracked manifest, relevant untracked files, and every file hash.
7. Guard verifies its parent directly matches the recorded live Paru PID/start time/executable (the POSIX `exec` replaces Paru's mandatory `sh -c` process), and owner UID matches.
8. In `BEGIN IMMEDIATE`, guard atomically changes exactly one matching unexpired approval `armed → claimed`. Missing, duplicate, expired, unexpected pkgbase, or changed identity aborts.
9. Guard recomputes the complete no-follow manifest after the atomic claim and requires it to equal both the approved digest and the pre-claim digest, then records verification digest/time and exits `0`. On any error it marks the approval invalid and exits a dedicated nonzero guard code.
10. Wrapper observes Paru/build outcome. A successful exact build changes `claimed → consumed`; cancellation/failure invalidates every remaining `armed` or `claimed` approval. A claimed approval is never reusable by a later Paru process.
11. A cached package may be reused only when its SHA-256 is already linked to a successful build of this exact identity. Otherwise execution forces rebuild.

### Accepted-specification conflict: hook timing

The accepted specification currently says `PreBuildCommand` verifies "immediately before each build." Paru 2.1 does not provide that timing: it runs all package-base hooks together, then can perform later non-recipe work before individual makepkg calls.[4] This must not be silently redefined.

- **Option A — amend the guarantee to the actual security boundary:** the hook is the final complete recipe-content verification before Paru invokes any makepkg/recipe code; `--skipreview` removes Paru's later edit path, and the manifest is computed twice around the atomic claim. Repository dependency installation may occur later. **Recommended**, because no hostile recipe code runs in the interval and the remaining same-UID race is already explicit.
- **Option B — require an upstream Paru per-build hook:** preserves the literal wording, but blocks production on an upstream change and still cannot defeat a malicious same-UID process after return.
- **Option C — interpose an AURoscope makepkg proxy:** can reverify immediately before every makepkg invocation, but becomes a materially broader process boundary, must handle multiple source/prepare/build invocations and VCS `pkgver()` mutations, and needs separate chroot handling. Defer unless Mathieu rejects Option A.

**Decision:** Option A was accepted on 2026-08-30. The guard is the final complete recipe-identity verification after AURoscope review and before any recipe-supplied code executes; it is not represented as adjacent to each build. See [`ADR-0005`](../decisions/0005-recipe-identity-guard-boundary.md).

### Limits and residual risk

- Paru runs all pre-build commands before its own review and before later build steps, not literally immediately before each makepkg exec.[4] `--skipreview` removes the known post-guard edit path.
- A separate process running as the same Unix user can mutate files after guard return. No user-space hook can eliminate that adversary; the guarantee is against hostile repository content before its code executes and accidental/concurrent drift, not a compromised user account.
- Once makepkg sources the approved PKGBUILD, that code can modify its own worktree. This is execution of the approved program, not pre-execution substitution; a clean build sandbox remains a separate deferred defense.
- Package cache authenticity requires artifact hashes, not recipe hashes.

## 6. Deterministic scanner and LLM contracts

### Deterministic finding schema v1

```json
{
  "schema_version": 1,
  "finding_id": "sha256:...",
  "source": "deterministic",
  "rule_id": "download-piped-to-shell",
  "rule_version": 1,
  "severity": "critical",
  "path": "escaped-display/PKGBUILD",
  "line": 47,
  "byte_range": [1234, 1271],
  "evidence": "curl -fsSL ... | bash",
  "evidence_sha256": "...",
  "message": "Downloaded bytes are executed by a shell"
}
```

A finding is immutable. Aggregate signal is computed separately. Stable `finding_id` hashes rule/version/location/evidence digest, not human prose.

### Initial rule catalogue

1. collection integrity: symlink, special file, escaping path, unreadable file, duplicate/case-confusable path, oversized/truncated content, untracked relevant file;
2. execution/download: curl/wget-to-shell, eval/dynamic shell, decoded execution, runtime network in build/install functions;
3. integrity: `SKIP`, HTTP source, missing/changed checksum arrays, signature bypass, mutable VCS ref;
4. privilege/persistence: sudo/su, setuid/setgid, file capabilities, world-writable modes, writes to sudoers/polkit/PAM, systemd units/timers, udev, tmpfiles, sysusers, kernel/modules/boot;
5. data access: SSH/GPG/browser/cloud credentials, broad `$HOME` traversal, secret environment expansion;
6. package semantics: new/changed `.install`, `provides`, `conflicts`, `replaces`, install paths, hooks, maintainer/source domains;
7. obfuscation: base64/xxd decode, generated commands, long encoded blobs, control/bidi characters.

Regex-only rules are insufficient for shell semantics. V1 may combine lexical patterns with a shell parser only if that parser is separately justified and never executes input. Findings must say what was observed, not claim malicious intent.

### Context and truncation

- Always include manifest, metadata, deterministic findings, changed files, unified diff, and full context for executable recipe files within limits.
- Default proposal: 256 KiB/file, 2 MiB aggregate text, 512 KiB diff, 200 files. These are starting limits to benchmark, not accepted constants.
- Binary files get metadata/hash/type only unless a dedicated parser exists.
- Truncation or omitted relevant files sets inspection `partial`, advisory `unknown|caution`, and always pauses for human review. It never silently becomes clear.
- First install compares against an empty baseline and includes full relevant text. Later inspection uses last approved/built identity plus complete changed-file context.

### LLM request/response

Use an inference-only HTTP adapter. Do not grant tools, shell, filesystem, browser, plugins, or agent loops. Send only selected recipe data; strip host absolute paths, usernames, environment, credentials, and unrelated comments by policy. Package content is serialized inside a JSON data object and labelled untrusted; prompt-injection text remains data, never policy.

Response v1 contains only:

```text
contract_version
advisory_level: informational|caution|high|unknown
summary (bounded)
change_explanations[]
attention_items[] {path, line?, severity, observation, uncertainty}
uncertainties[]
```

It contains no `approve`, `allow`, `deny`, or final decision field. Local validation requires known paths, real line ranges, bounded strings/counts, no extra fields, and valid UTF-8. Deterministic findings are supplied read-only and are not echoed as authoritative replacements.

### Provider/config/privacy options

- **Provider SDK/agent CLI:** feature-rich but expands behavior and credential surface. Not recommended as default.
- **OpenAI-compatible HTTP:** small, inference-only, explicit endpoint/model/key environment name. Recommended.
- **Disabled LLM:** supported; inspection may still be complete from deterministic scanning, with advisory level derived without pretending model evidence exists.

Default failure behavior: timeout, transport error, schema failure, truncation, or provider refusal records LLM status `failed`, overall inspection `partial` only if model analysis was required by policy, advisory `unknown`, and pauses for human decision. Human approval remains possible after visible failure in interactive human-authority mode; failure is neither veto nor approval.

Log request/response hashes and bounded redacted metadata. Raw prompts/responses are `0600`, retained by explicit policy, and omitted entirely when privacy mode disables them. Secrets are referenced by environment-variable name, never stored in TOML or SQLite.

### Historical reference used narrowly

From `2027a/paru-llm-audit`, retain only proven ideas/fixtures: tracked-artifact inventory, no-follow collection, escaped terminal output, visible truncation findings, prompt-injection fixtures, strict model output validation, and before/after snapshot comparison.[24]

Do **not** inherit its hook-centered architecture, aggregate automatic allow/block policy, Codex CLI backend, Python code, or mutable acceptance state. AURoscope has a different human-authority and transaction model.

## 7. Configuration, XDG, reports, and cleanup

The XDG Base Directory specification assigns configuration, durable state, cache, and runtime files distinct roots.[14]

### Proposed exact layout

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

If `XDG_RUNTIME_DIR` is set, require an absolute path owned by the user with safe permissions. Otherwise create the fallback with `os.MkdirTemp`; never use predictable names. Start with umask `0077`, create all AURoscope roots as `0700`, and verify SQLite's `-wal`/`-shm` auxiliary files remain private. Refuse config/state/cache roots that are relative, symlinks at sensitive final components, not owned by the user, or group/world writable.

### Config proposal

TOML with explicit sections: `[policy]`, `[paths]`, `[scanner]`, `[llm]`, `[review]`, `[retention]`, `[compatibility]`. Unknown keys are errors. Defaults live in code; no required config file. CLI overrides are explicit and recorded in the transaction. Arrays represent commands (`viewer = ["nvim", "-f"]`) so no shell parsing is required.

`$VISUAL`, then `$EDITOR`, then pager fallback applies when no configured argv exists. Parse the selected environment value with `mattn/go-shellwords` while explicitly disabling environment-variable and backtick expansion; require at least one argv element, resolve the executable without elevation, and append the report path as a new argv element. Never invoke `/bin/sh -c` or interpolate the path into command text.[27]

### Reports

- Render reports from SQLite plus immutable recipe blobs; reports are not read back.
- Atomic write: private temp file in destination directory, `fsync`, rename, directory `fsync` where supported.
- Escape control characters, bidi controls, invalid UTF-8, and terminal sequences.
- `$VISUAL`/`$EDITOR` receives a read-only generated report, not the build workspace. “Inspect recipe” opens a private copy or pager; direct edits require an explicit edit workflow followed by rehash/reinspection.

### Cleanup protocol

- Every transaction owns one recorded work root and a marker containing transaction ID, owner UID, creation time, and random nonce; marker data is not authority by itself.
- `defer` cleanup immediately after child exit; signal handler requests cancellation, while normal unwinding performs deletion.
- Remove directory contents using descriptor-relative, no-follow traversal; never `RemoveAll` a path reconstructed solely from package text or marker content.
- On startup and `auroscope cleanup`, enumerate only direct children of the configured AURoscope runtime/cache root, cross-check SQLite ownership, PID+start time, age, device, inode, and UID, then delete.
- Default stale runtime grace: 24 hours. Active recorded process sessions are never removed.
- Cache pruning is quota+age based, uses a lock, and never follows symlinks. Reports/state are not cache.

## 8. Threat model

### Assets

- root-owned system and Pacman database;
- user files, secrets, model credentials, and terminal integrity;
- exact recipe approval and audit history;
- trusted official repository transaction semantics;
- package artifacts installed by Pacman.

### Actors and trust boundaries

- human user: authority, but may make mistakes;
- AUR maintainer/repository and all tracked/untracked content: hostile;
- upstream source URLs/archives: hostile and mostly out of v1 inspection scope;
- Paru/Pacman/makepkg/Git: trusted versioned dependencies with constrained contracts;
- LLM/provider: untrusted advisory processor and privacy boundary;
- other same-UID process: can race local files; partially mitigated, not fully defeatable;
- root/sudo boundary: Pacman alone receives elevation in the normal flow.

### Abuse cases and mitigations

| Threat | Required mitigation | Residual risk/proof |
|---|---|---|
| symlink/path traversal | Git-tree paths as bytes; descriptor-relative no-follow open; reject special/escaping paths | integration fixture with intermediate symlink replacement |
| prompt injection | JSON data envelope, no tools, strict local validation, no decision field | adversarial prompt corpus; model can still give poor advice |
| recipe changed after review | commit+tree+manifest+per-file hashes, workspace dev/inode, one-shot guard | same-UID post-guard race remains |
| Paru review edits after guard | execution uses `--skipreview` | contract test against supported Paru versions |
| stale/replayed approval | transaction/process binding, expiry, atomic claim, terminal invalidation | `/proc` unavailable is fail closed |
| cached binary substitution | exact artifact SHA-256 linked to successful build identity; otherwise rebuild | compiler/build compromise out of scope |
| hostile filename/terminal escape | byte-safe paths, escaped rendering, bounded output | golden tests with control/bidi/invalid UTF-8 |
| PATH/config/editor manipulation | resolve configured executables before elevation, absolute guard command, owner/mode checks, argv arrays | user's own trusted config can choose unsafe tools |
| sudo leakage | wrapper never runs as root; no secrets in environment passed to Pacman/build; sanitize model vars | PKGBUILD runs as user and can read that user's files unless build sandbox added |
| Pacman concurrency | application mutator lock plus Pacman's own `db.lck`; never delete lock | real concurrent-container test |
| cleanup escape | owned roots, DB cross-check, descriptor-relative deletion | crash between create and DB record leaves age-gated residue |
| Git helper execution | fixed absolute Git, disable hooks, external diff, textconv, filters, credential prompts; do not checkout untrusted symlink targets | Git vulnerabilities remain dependency risk |
| partial official upgrade | complete repo-only `-Syu` phase; never convert held AUR into held repo packages | AUR may break temporarily after upgrade |
| LLM data exfiltration | minimal selected context, no host paths/secrets, explicit provider, privacy mode | recipe content itself is sent when enabled |
| database corruption | WAL, FULL sync for approvals, quick_check, backup before migration | disk/hardware failure needs user backup |

## 9. Risk-proportional test strategy

### Unit

- raw argv classifier matrix and unknown-option preservation;
- Paru `--order` record parser, unknown record rejection, selected-name validation;
- canonical manifest with byte filenames, modes, symlinks, size/limits;
- each deterministic rule with positive, negative, obfuscation, and false-positive fixtures;
- LLM decoder: extra fields, invalid enum, fake path/line, oversized text, decision language;
- lifecycle transitions and exit-category mapping;
- XDG resolution, ownership/mode validation, cleanup eligibility;
- report escaping and deterministic rendering.

### Property/fuzz

- argv classifier never changes transparent argv;
- path parser/manifest cannot escape root or panic on arbitrary bytes;
- Paru record and LLM JSON parsers reject malformed/oversized input without excessive allocation;
- scanner never suppresses deterministic findings under arbitrary LLM output;
- lifecycle model admits no `approve` without an exact recipe identity, a recorded inspection status (including `partial` or `failed`), visibly recorded limitations/errors, and an explicit human decision;

### SQLite/migration

- fresh schema, each forward migration, interrupted migration, checksum mismatch;
- foreign keys, uniqueness, atomic approval claim under concurrent goroutines/processes;
- WAL busy timeout, reader during writer, crash after claim/before consume;
- quick_check, backup/restore, retention without deleting referenced identities;
- run the suite against the selected driver and SQLite version.

### Executable dependency contracts

For every supported Paru/Pacman package combination in a disposable Arch image:

- bare Paru maps to `-Syu`;
- interactive search menu stays on terminal stream, selected names alone on captured stdout, selected and cancelled exit behavior recorded;
- `-P --order` grammar for repo/AUR target/dep/make and missing/conflict cases;
- `PreBuildCommand` cwd, env, call count/order, already-built behavior, failure propagation;
- `--skipreview` prevents post-guard editing;
- cached artifact policy forces rebuild or verifies an exact recorded artifact;
- pure repo commands preserve output, TTY, signals, and exit status;
- Pacman `--print-format` origin/plan fields and `db.lck` contention.

### Security integration

- mutate commit/file/mode/symlink/untracked relevant file after approval;
- replace workspace directory inode;
- replay approval from another Paru PID, after expiry, after cancellation, and twice concurrently;
- race mutation during hashing and immediately after claim;
- malicious config/PATH/editor/pager/environment;
- prompt injection, fake deterministic finding, model tool-call-like output, timeout, schema bomb;
- crash at each approval transition and startup recovery;
- verify no package code executes during collector/scanner/model phases with marker fixtures.

### End-to-end

Use disposable Arch containers/VMs with private Pacman DB/root and synthetic local repositories/AUR Git remotes. Never target the real workstation.

1. repo-only install/query/remove pass-through;
2. explicit mixed repo+AUR target with dependencies;
3. bare upgrade with complete repo phase and approved/deferred/rejected AUR phase;
4. native interactive search selection;
5. first install/full review and later differential review;
6. scanner clear/noteworthy/partial/failed plus LLM enabled/disabled/failed;
7. cancellation at selection, model, editor, approval, guard, build, Pacman confirmation;
8. successful build, recorded artifact reuse, tampered cache rejection;
9. interrupted migration and stale-work cleanup;
10. package build/install/remove of AURoscope itself, `namcap`, reproducibility attempt.

### Quality gates before implementation can be called complete

- supported-version contract matrix green;
- race detector and fuzz smoke green;
- no collector path executes/sources PKGBUILD (marker tests);
- anti-TOCTOU negative tests all stop before makepkg;
- complete official upgrade proof in disposable Arch;
- clean install/upgrade/rollback of the AURoscope package;
- independent review of schema migrations, path handling, process/signal code, and approval protocol.

## 10. Unresolved risks and explicit non-goals

- Current stable Paru does not satisfy clean native selection capture. ADR-0002 permits a temporary fail-closed 2.1.0 parser, but format and locale drift can disable that adapter; the clean durable contract still depends on a stable release containing `d1dfbc4`.
- Paru lacks a stable JSON plan API; its documented text grammar must be contract-tested and version-gated.
- Transaction-specific config layering is grounded in Paru's parser, but still needs an executable spike for nested includes, repeated sections, whitespace paths, and preservation of every supported user option before ADR acceptance.
- Same-UID post-guard mutation cannot be fully prevented without moving execution to an immutable/sandboxed snapshot; v1 must state this limit.
- Upstream source archives and compiled binary behavior remain out of scope; approval says “reviewed recipe,” never “safe software.”
- Clean chroot/sandbox builds are valuable but remain deferred unless Mathieu expands scope.
- Non-interactive autonomous approve/reject policy remains deferred and requires a separate ADR.

## 11. Decision set for Mathieu

Each decision is tracked in a dedicated Forgejo issue containing its context, evidence, alternatives, trade-offs, recommendation, risks, and expected reply:

| Decision | Issue | Status |
|---|---|---|
| D1 — Paru compatibility | [#2](https://git.2027a.net/2027a/auroscope/issues/2) | **Accepted:** [ADR-0002](../decisions/0002-paru-native-selection-compatibility.md) |
| D2 — orchestration | [#3](https://git.2027a.net/2027a/auroscope/issues/3) | **Accepted:** [ADR-0003](../decisions/0003-multi-stage-paru-orchestration.md) |
| D3 — upgrades | [#4](https://git.2027a.net/2027a/auroscope/issues/4) | Proposed |
| D4 — TOCTOU timing | [#5](https://git.2027a.net/2027a/auroscope/issues/5) | **Accepted:** [ADR-0005](../decisions/0005-recipe-identity-guard-boundary.md) |
| D5 — Go/process architecture | [#6](https://git.2027a.net/2027a/auroscope/issues/6) | Proposed |
| D6 — SQLite driver | [#7](https://git.2027a.net/2027a/auroscope/issues/7) | **Accepted:** [ADR-0007](../decisions/0007-mattn-go-sqlite3-cgo.md) |
| D7 — remaining Go dependencies | [#8](https://git.2027a.net/2027a/auroscope/issues/8) | **Accepted:** [ADR-0008](../decisions/0008-minimal-direct-go-dependencies.md) |
| D8 — SQLite state model | [#9](https://git.2027a.net/2027a/auroscope/issues/9) | Proposed |
| D9 — approval protocol | [#10](https://git.2027a.net/2027a/auroscope/issues/10) | Proposed |
| D10 — scanner/LLM contracts | [#11](https://git.2027a.net/2027a/auroscope/issues/11) | Proposed |
| D11 — XDG/cleanup/retention | [#12](https://git.2027a.net/2027a/auroscope/issues/12) | Proposed |
| D12 — threat model/tests | [#13](https://git.2027a.net/2027a/auroscope/issues/13) | Proposed |
| D13 — local PKGBUILD scope | [#14](https://git.2027a.net/2027a/auroscope/issues/14) | Proposed |

Workflow for every decision issue:

1. The issue body starts with the exact first-line marker `MODE: DECISION`; the watcher keeps it under `Agent/Human` and never starts implementation.
2. Mathieu replies with his chosen option, amendment, question, or rejection. Each new ordinary comment creates only a bounded scratch discussion task: Héphaïstos may answer in the issue but may not edit the repository, create an ADR/PR, close the issue, or implement code.
3. When the latest concrete proposal is accepted, Mathieu posts a comment whose entire trimmed content is exactly `GO DECISION`. The watcher accepts it only from Mathieu's pinned Forgejo login and numeric user ID, and only when it is the latest external comment.
4. The watcher then transfers responsibility to `Agent/Hermes` and creates an idempotent decision-finalization task. Only then may Héphaïstos record the decision in an ADR, update design/specification documents, verify the remote commit/PR, comment the evidence, and close the issue.
5. `GO DECISION` never authorizes production implementation or the implementation plan. Any resulting work requires a separate issue starting with `MODE: EXECUTION`.

Issues without an exact mode marker fall back to a single existing routing label; missing or conflicting routing is classified `Agent/Needs Review` and executes nothing. Until step 3 is complete and the resulting ADR is committed for a given decision, that item remains **Proposed**. Accepted items below link to their ADR. The PR itself is evidence and discussion material, not approval.

Please accept, amend, reject, or defer each item. Recommendations are not yet decisions.

1. **D1 — Paru compatibility — Accepted in [ADR-0002](../decisions/0002-paru-native-selection-compatibility.md):** retain the first stable Paru release containing `d1dfbc4` as the durable floor; meanwhile permit an isolated, temporary, fail-closed adapter for verified 2.1.0 output. Detect capability behaviorally and remove the adapter after a fixed stable release passes the contract matrix. A pinned post-fix commit remains design/test-only.
2. **D2 — orchestration — Accepted in [ADR-0003](../decisions/0003-multi-stage-paru-orchestration.md):** adopt two-stage Paru planning/review/execution; do not parse Paru's human UI beyond ADR-0002's temporary 2.1.0 exception, and never replace its resolver.
3. **D3 — upgrades:** review a provisional AUR plan before mutation, execute a complete repository-only `-Syu` phase, then re-plan/revalidate before the independently deferrable AUR phase. **Recommended: accept.**
4. **D4 — execution/cache and specification amendment — Accepted in [ADR-0005](../decisions/0005-recipe-identity-guard-boundary.md):** treat the guard as the final complete recipe-identity verification after AURoscope review and before any recipe-supplied code executes, without claiming that it is adjacent to each build; run with `--skipreview`, and rebuild unless a cached artifact hash is tied to the exact approved identity.
5. **D5 — Go process architecture:** one executable, explicit internal packages, standard-library argv/process handling, no CLI framework and no PTY dependency initially. **Recommended: accept.**
6. **D6 — SQLite driver — Accepted:** use `mattn/go-sqlite3 v1.14.50` with CGO for Arch `linux/amd64` v1; revisit pure Go only with real cross-target need. Record: [ADR-0007](../decisions/0007-mattn-go-sqlite3-cgo.md).
7. **D7 — remaining dependencies — Accepted:** TOML via `pelletier/go-toml/v2`, `$VISUAL` parsing via `mattn/go-shellwords v1.0.14` with environment and backtick expansion disabled, embedded SQL migrations, typed local LLM validation, and no LLM SDK/framework or PTY dependency without demonstrated need. See [ADR-0008](../decisions/0008-minimal-direct-go-dependencies.md).
8. **D8 — state model:** normalized immutable evidence/history plus explicit transaction/approval/build lifecycle tables; WAL, short writes, application mutator lock, no event-sourcing framework. **Recommended: accept.**
9. **D9 — approval protocol:** transaction/process/workspace-bound one-shot approvals, 30-minute default expiry, atomic claim, all terminal states invalidate leftovers, explicit same-UID residual risk, and human approval remains representable after visibly recorded partial/failed analysis. **Recommended: accept.**
10. **D10 — scanner/LLM:** immutable versioned deterministic findings; inference-only optional LLM with no decision field/tools; model failure pauses for human but does not autonomously veto. **Recommended: accept.**
11. **D11 — XDG/cleanup:** exact layout and permissions above, private runtime dirs, 24-hour stale-work recovery, 90-day reports, 30-day/1-GiB reconstructible cache defaults. **Recommended: accept the structure; amend numerical retention defaults if desired.**
12. **D12 — threat/tests:** adopt the stated threat boundary and require real supported-version, race, TOCTOU, and disposable-Arch integration proof before v1. **Recommended: accept.**

13. **D13 — local PKGBUILD builds/repositories:** reject `-B`, targetless `-U`, local path targets, modes containing `pkgbuilds`, and any unexpected PKGBUILD record in v1 because Paru may execute `makepkg --printsrcinfo` before the guard; normalize allowed user modes with final trusted reset flags and design safe local-recipe support separately. **Recommended: accept.**

## Sources

[1] https://github.com/Morganamilo/paru/blob/v2.1.0/man/paru.8
[2] https://github.com/Morganamilo/paru/blob/v2.1.0/man/paru.conf.5
[3] https://github.com/Morganamilo/paru/blob/70f66dc9eddb40e264ee6c9197541262b7792c9c/src/lib.rs
[4] https://github.com/Morganamilo/paru/blob/70f66dc9eddb40e264ee6c9197541262b7792c9c/src/install.rs
[5] https://github.com/Morganamilo/paru/blob/70f66dc9eddb40e264ee6c9197541262b7792c9c/src/util.rs
[6] https://github.com/Morganamilo/paru/commit/d1dfbc48895a2a3b3fadc14569c3f2ffb5112b2e
[7] https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/util.rs
[8] https://aur.archlinux.org/cgit/aur.git/tree/PKGBUILD?h=paru&id=329be2113c590046cb29858c23d9b96a8d7bd586
[9] https://gitlab.archlinux.org/archlinux/packaging/packages/pacman/-/blob/abdc0dfedf3ca553a02dde8551e972fe745535b7/PKGBUILD
[10] https://gitlab.archlinux.org/pacman/pacman/-/blob/v7.1.0/doc/pacman.8.asciidoc
[11] https://gitlab.archlinux.org/pacman/pacman/-/blob/v7.1.0/doc/makepkg.8.asciidoc
[12] https://gitlab.archlinux.org/pacman/pacman/-/blob/v7.1.0/scripts/makepkg.sh.in
[13] https://gitlab.archlinux.org/pacman/pacman/-/blob/v7.1.0/doc/makepkg.conf.5.asciidoc
[14] https://specifications.freedesktop.org/basedir-spec/latest
[15] https://www.sqlite.org/wal.html
[16] https://www.sqlite.org/foreignkeys.html
[17] https://www.sqlite.org/lang_transaction.html
[18] https://pkg.go.dev/os/exec
[19] https://pkg.go.dev/os/signal
[20] https://github.com/mattn/go-sqlite3
[21] https://pkg.go.dev/modernc.org/sqlite
[22] https://github.com/creack/pty
[23] https://github.com/pelletier/go-toml
[24] https://git.2027a.net/2027a/paru-llm-audit
[25] https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/config.rs
[26] https://github.com/Morganamilo/paru/blob/9ac3578807a87858651e81a02586ceb947686e7c/src/order.rs
[27] https://github.com/mattn/go-shellwords
