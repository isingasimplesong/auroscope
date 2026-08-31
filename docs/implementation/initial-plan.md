# AURoscope initial implementation plan

**Status:** Proposed for implementation sequencing review under [execution issue #17](https://git.2027a.net/2027a/auroscope/issues/17). This document is a plan, not authorization to write production code.

**Design baseline:** `main` at merge commit `31e73bd3f63d7e4a6a1d21061f719e3f7703a541`, with [ADR-0001](../decisions/0001-go-and-self-hosted-arch-packaging.md) through [ADR-0014](../decisions/0014-reject-local-pkgbuild-inputs-in-v1.md) accepted.

## 1. Goal and non-negotiable boundaries

Deliver a first useful Arch Linux `linux/amd64` release that can replace bare Paru for the supported surface while keeping Paru/Pacman authoritative, reviewing exact AUR recipes before recipe-supplied code executes, and leaving every consequential interactive decision to the user.

The implementation must preserve these boundaries throughout the sequence:

- package content, metadata, filenames, diffs, fixtures, and model input are hostile data, never instructions;
- inspection never executes or sources `PKGBUILD`, package source material, or any recipe-supplied file;
- official repository upgrades happen first as one complete native Paru/Pacman phase; AUR planning and review happen only after it succeeds;
- Paru remains selector, resolver, builder, and Pacman frontend; AURoscope never grows a second resolver;
- deterministic findings, aggregate advisory signal, LLM assessment, technical status, and human decision remain distinct;
- only an explicit human `approve` decision may arm a one-shot approval for the exact identity reviewed;
- a `clear` advisory signal may make review concise or batched, but it never creates approval automatically: every recipe handed to the guard still needs an append-only explicit human `approve` decision, and Paru/Pacman's later native confirmation remains intact;
- local PKGBUILD operations and configured PKGBUILD-repository records remain rejected in v1;
- a fresh pre-execution plan comparison narrows drift but is not atomic with Paru's subsequent resolver run; implementation must prove a supported observation/guard surface or stop for a design amendment rather than claim complete closure equality;
- package artifact reuse stays disabled unless a supported post-build surface proves the exact files and hashes without invoking unsafe makepkg introspection or parsing unrestricted human output;
- no implementation slice may claim sandboxing, local-host protection, package safety, or per-build-adjacent guard timing;
- the real workstation Pacman database, root, cache, and package state must never be used by automated integration or end-to-end tests.

Any implementation finding that contradicts an accepted ADR stops the affected slice. It requires a separate `MODE: DECISION` issue and an accepted amendment or superseding ADR; it must not be “fixed” silently in code.

## 2. Delivery and branch policy

This planning PR contains documentation only. After Mathieu accepts it:

1. Before Slice 00, require a `MODE: EXECUTION` issue in which Mathieu explicitly advances the repository from design to implementation; the marker alone is not a phase transition. That first implementation PR updates the phase statements in `AGENTS.md`, `README.md`, and `docs/design-phase.md` without rewriting accepted design history.
2. Create one Forgejo issue per numbered slice. Its first non-empty line must be `MODE: EXECUTION` and its body must link this plan, list the exact slice, prerequisites, and acceptance gates.
3. Create `execution/issue-<number>-<slug>` from current `origin/main` only after prerequisites are merged.
4. Keep one coherent slice per branch and PR. Do not stack later slices on an unmerged feature branch unless Mathieu explicitly authorizes it.
5. Use strict RED → GREEN → REFACTOR commits or preserve that trace in the PR description when a coherent commit cannot remain red.
6. Push signed commits with the Hephaistos identity. Open a Forgejo PR against `main`; agents do not merge their own PRs.
7. Rebase or merge current `origin/main` before final verification according to repository policy; never force-push by default.
8. A PR is reviewable only when the slice-specific commands and the cumulative baseline pass. A green unit suite never substitutes for a required live contract or disposable-Arch gate.
9. Update this plan only when dependencies, scope, or acceptance criteria materially change. Record the reason and affected later slices.

Recommended PR size is one numbered slice. Slices 00 and 01 may be combined only if their combined diff remains bootstrap-sized; the security proof, guard protocol, upgrade workflow, and release gate must remain separate review units.

## 3. TDD and verification policy

Every slice follows the same loop:

1. **RED:** add the smallest failing unit, fuzz, contract, integration, or end-to-end test that states the next accepted behavior. Record the observed failure in the PR description.
2. **GREEN:** implement only enough production behavior to pass that test without broad speculative abstractions.
3. **REFACTOR:** remove duplication, tighten names and boundaries, then rerun the focused test and cumulative baseline.
4. **VERIFY:** execute the slice commands on the required environment and preserve relevant logs as CI artifacts when a live dependency is involved.

The numbered slices are mergeable delivery envelopes, not permission to implement every RED clause in one batch. Within a slice, treat each separately stated behavior as an ordered micro-slice: write one focused failing test, observe the intended failure, make only that test pass, refactor while green, and commit before advancing. Split any clause that cannot remain a small reviewable change into named subtests/subcommits inside the same execution issue.

Cumulative local baseline after Go bootstrap:

```console
CGO_ENABLED=1 go test ./...
CGO_ENABLED=1 go test -race ./...
go vet ./...
test -z "$(git ls-files -co --exclude-standard -- '*.go' | xargs -r gofmt -l)"
git diff --check
```

Additional rules:

- Fuzz targets must ship with deterministic seed corpora and bounded input sizes. CI runs seed regression on every PR; scheduled/release CI runs time-bounded fuzzing.
- Tests that spawn commands use argv arrays and controlled temporary directories. Shell scripts are confined to test/packaging harnesses and must not interpolate hostile fixture text.
- Live Paru/Pacman/makepkg tests run only in an ephemeral Arch container or disposable VM. They must require an explicit private Pacman root/database and abort if `/`, `/var/lib/pacman`, or the host cache would be targeted.
- Network-dependent contract tests pin exact upstream commits/package versions and are separate from deterministic unit tests.
- Race tests do not make SQLite or the approval protocol secure against a hostile local account; they verify ordinary correctness only.

## 4. Planned repository layout

The sequence introduces only files needed by implemented behavior:

```text
cmd/auroscope/
internal/approval/
internal/cli/
internal/config/
internal/llm/
internal/paru/
internal/process/
internal/recipe/
internal/report/
internal/review/
internal/scanner/
internal/store/
internal/store/migrations/
internal/testutil/
testdata/recipes/
testdata/paru/
testdata/llm/
test/contract/
test/integration/
test/e2e/
packaging/arch/
scripts/
.gitea/workflows/
```

Test files live beside their package unless they require a real dependency or process boundary, in which case they live under `test/contract`, `test/integration`, or `test/e2e`. New package-level interfaces are allowed only for clock/randomness, process execution, model transport, and store transactions when a test needs the boundary.

## 5. Dependency-ordered implementation slices

### 00 — Reproducible Go and CI bootstrap

**Prerequisites:** accepted plan; a dedicated execution issue containing Mathieu's explicit phase advance; clean `main`; Arch `linux/amd64` target.

**Files:**

- create `go.mod`, `go.sum`;
- create `cmd/auroscope/main.go`, `cmd/auroscope/main_test.go`;
- create `internal/testutil/testutil.go`;
- create `scripts/check.sh`;
- create `scripts/check-docs.sh`;
- create `.gitea/workflows/verify.yml`;
- update `AGENTS.md`, `README.md`, and `docs/design-phase.md` to record the authorized implementation phase; add only verified developer commands.

**RED:** add tests for deterministic build metadata formatting and for an injectable command entry returning an exit code rather than calling `os.Exit` below `main`. Make `scripts/check-docs.sh` fail on broken repository-relative links, malformed Markdown, stale phase statements, and missing ADR/specification traceability using pinned documented tooling.

**GREEN:** establish module `git.2027a.net/2027a/auroscope`, a minimal command entry, `-trimpath` build flags, version/commit injection variables, and a CI job using Arch with CGO enabled. Pin Go/tool images or package snapshots explicitly.

**REFACTOR:** keep `main` wiring-only; no CLI framework or domain packages created pre-emptively.

**Verification:** cumulative baseline; `scripts/check-docs.sh`; `CGO_ENABLED=1 go build -trimpath ./cmd/auroscope`; inspect embedded module/VCS metadata with `go version -m` in the Arch CI image. Do not claim AURoscope's own `--version`: the accepted classifier preserves Paru's transparent `--version` behavior.

**Exit criteria:** signed Arch-targeted binary builds; CI proves CGO toolchain availability; no runtime or package-management behavior is claimed yet. Covers ADR-0001 and prepares ADR-0007.

### 01 — Raw argv classifier and fail-closed v1 boundary

**Prerequisites:** slice 00.

**Files:**

- create `internal/cli/classify.go`, `internal/cli/classify_test.go`, `internal/cli/classify_fuzz_test.go`;
- create `internal/cli/command.go`, `internal/cli/command_test.go`;
- create `testdata/paru/argv-cases.json`;
- modify `cmd/auroscope/main.go` only to call the classifier/dispatcher.

**RED:** table and fuzz tests for every class in the design argument table: byte-for-byte pass-through, intercepted searches/installs/upgrades, explicit `-U` archive pass-through, `--noconfirm` rejection when a new AUR decision is possible, unknown/ambiguous fail-closed behavior, and pre-launch rejection of `-B`, targetless `-U`, path-like targets, `file:`, and modes containing `pkgbuilds`/`p`.

**GREEN:** parse only AURoscope-owned subcommands and the minimum Pacman/Paru operation grammar; retain the original `[]string` unchanged for transparent flows; produce typed classification/rejection reasons and final trusted mode-reset intent without starting a child.

**REFACTOR:** separate syntactic classification from execution policy; do not normalize unknown options or build a general Paru parser.

**Verification:** `go test ./internal/cli -run .`; `go test ./internal/cli -fuzz=FuzzClassify -fuzztime=30s`; cumulative baseline.

**Exit criteria:** no rejected or ambiguous local-recipe input can launch Paru; every transparent fixture returns the exact original argv. Covers ADR-0006 and ADR-0014.

### 02 — XDG paths, strict configuration, and private runtime roots

**Prerequisites:** slice 00; classifier types from slice 01.

**Files:**

- create `internal/config/config.go`, `internal/config/config_test.go`;
- create `internal/config/paths.go`, `internal/config/paths_linux_test.go`;
- create `internal/config/editor.go`, `internal/config/editor_test.go`;
- create `internal/config/testdata/*.toml`;
- modify `go.mod`, `go.sum` to pin reviewed `github.com/pelletier/go-toml/v2` and `github.com/mattn/go-shellwords v1.0.14`.

**RED:** tests for optional config/defaults; unknown TOML keys; explicit backend modes; secret environment-variable references; exact XDG fallbacks; absolute/owned/non-symlink/private roots; `0077` umask effects; `0600`/`0700` modes; safe `XDG_RUNTIME_DIR` and unpredictable `MkdirTemp` fallback; configured viewer argv; `$VISUAL` → `$EDITOR` → pager precedence; disabled environment/backtick expansion; report path appended as a distinct argv element.

**GREEN:** implement typed config sections `[policy]`, `[paths]`, `[scanner]`, `[llm]`, `[review]`, `[retention]`, `[compatibility]`, strict decoding, path validation, and private directory/file creation. Do not create persistence tables or execute viewers yet.

**REFACTOR:** centralize ownership/mode checks and keep package-derived strings out of path authority.

**Verification:** `go test ./internal/config`; dependency license/module inventory; cumulative baseline.

**Exit criteria:** all configured roots match ADR-0012; secrets are never accepted as TOML values; command parsing never invokes a shell. Covers ADR-0008, ADR-0011, and ADR-0012.

### 03 — Linux process runner, terminal preservation, and cancellation

**Prerequisites:** slices 00–02.

**Files:**

- create `internal/process/runner.go`, `internal/process/runner_test.go`;
- create `internal/process/result.go`, `internal/process/result_test.go`;
- create `internal/process/group_linux.go`, `internal/process/group_linux_test.go`;
- create `test/integration/process_helper_test.go`.

**RED:** real-process tests proving argv-only execution, inherited stdin/stdout/stderr, optional stdout capture with inherited stderr, separate child exit code and terminating signal, child process-group creation, SIGINT/SIGTERM/SIGHUP forwarding, child wait before cleanup callback, and no implicit shell.

**GREEN:** implement a small runner around `os/exec` and Linux process groups with typed results. Make cancellation ordering explicit and injectable only where tests need it.

**REFACTOR:** isolate Linux-specific syscalls; avoid PTY and generic job-control abstractions.

**Verification:** `CGO_ENABLED=1 go test -race ./internal/process ./test/integration -run Process`; cumulative baseline.

**Exit criteria:** transparent execution can preserve child terminal/status semantics; cancellation always waits and retains exit/signal evidence. Covers ADR-0006. A PTY proposal is forbidden unless a later supported-version test fails because inherited descriptors are insufficient.

### 04 — Paru/Pacman/makepkg capability and grammar adapters

**Prerequisites:** slices 01 and 03.

**Files:**

- create `internal/paru/version.go`, `internal/paru/version_test.go`;
- create `internal/paru/capability.go`, `internal/paru/capability_test.go`;
- create `internal/paru/selection.go`, `internal/paru/selection_test.go`;
- create `internal/paru/order.go`, `internal/paru/order_test.go`, `internal/paru/order_fuzz_test.go`;
- create `internal/paru/commands.go`, `internal/paru/commands_test.go`;
- create fixtures under `testdata/paru/2.1.0/` and `testdata/paru/post-d1dfbc4/`;
- create `test/contract/paru_contract_test.go`, `test/contract/pacman_contract_test.go`, `test/contract/run-arch.sh`.

**RED:** tests for startup version parsing; behavioral stream capability; clean post-fix selection; bounded transitional 2.1.0 selection grammar; exit `1` success only with valid selected names; exit `1` empty cancellation; signal failure; line/count/control-character bounds; `INSTALL` versus `REPO` order records; AUR records; `MISSING`, unknown, conflicting, and `PKGBUILD` record rejection; final trusted mode-reset argv placement; Pacman repository confirmation; documented command exit behavior.

**GREEN:** implement versioned adapters and command builders only for exact supported contracts. Capability detection must use observed behavior, not version strings alone. Keep human-output parsing isolated to the temporary 2.1.0 adapter. The durable production floor remains the first stable Paru release containing `d1dfbc4`; the pinned post-fix commit is contract-test input, not a production minimum.

**REFACTOR:** share typed plan records without merging incompatible grammars; include a removal condition for the transitional adapter in code comments/tests.

**Verification:** unit/fuzz commands plus `test/contract/run-arch.sh` in ephemeral Arch against Paru `2.1.0-2`, the pinned post-fix reference, Pacman/makepkg `7.1.x`; cumulative baseline.

**Exit criteria:** adapter rejects every unproven grammar; native selection behavior and cancellation are executable facts; no contract test touches the host package database. Covers ADR-0002, ADR-0003, ADR-0004, and ADR-0014.

### 05 — SQLite bootstrap, migrations, and instrumental schema core

**Prerequisites:** slices 00 and 02.

**Files:**

- modify `go.mod`, `go.sum` to pin `github.com/mattn/go-sqlite3 v1.14.50`;
- create `internal/store/open.go`, `internal/store/open_test.go`;
- create `internal/store/migrate.go`, `internal/store/migrate_test.go`;
- create `internal/store/migrations/0001_identity_inspection.sql`;
- create `internal/store/lock.go`, `internal/store/lock_test.go`;
- create `internal/store/types.go`, `internal/store/store_test.go`;
- create `internal/store/backup.go`, `internal/store/backup_test.go`;
- create `test/integration/store_concurrency_test.go`.

**RED:** runtime failure/smoke for unusable non-CGO driver; exact supported SQLite library version and `PRAGMA compile_options` baseline; `foreign_keys=ON`; WAL, private `-wal`/`-shm`, busy timeout, and required synchronous behavior; ordered embedded migration checksums; empty/migrated/reopened DB; pre-migration backup/restore; `quick_check` failure; concurrent open/write behavior; one mutating application lock with concurrent read-only status access; reversible byte round trips; identity/inspection foreign-key constraints. Missing, added, or changed compile options fail compatibility until deliberately reviewed.

**GREEN:** add only `schema_migrations`, `package_bases`, `recipe_identities`, `recipe_files`, `inspections`, and `deterministic_findings`, because these are the first concrete identity/inspection workflows. Record the exact `go-sqlite3` build-tag policy and compile-option baseline for the supported Arch build; start with no optional feature tags unless a tested requirement justifies one. Exact columns require a concrete query or invariant. Add human decisions, transactions, approvals, process sessions, builds, artifacts, outcomes, and LLM assessments only in the later migration that delivers each corresponding feature.

**REFACTOR:** keep SQL forward-only and embedded; keep transactions short; no generic event/EAV/provenance/plugin schema.

**Verification:** `CGO_ENABLED=1 go test -race ./internal/store ./test/integration -run Store`; inspect SQLite compile options in Arch CI; backup/restore smoke; cumulative baseline.

**Exit criteria:** core invariants are enforced at SQL or short store-transaction boundaries; migration from empty and prior schema is deterministic; authoritative state refuses unsupported locking/filesystem behavior. Covers ADR-0007 and ADR-0009.

### 06 — Non-executing Git acquisition and canonical recipe identity

**Prerequisites:** slices 02, 03, and 05.

**Files:**

- create `internal/recipe/acquire.go`, `internal/recipe/acquire_test.go`;
- create `internal/recipe/manifest.go`, `internal/recipe/manifest_test.go`, `internal/recipe/manifest_fuzz_test.go`;
- create `internal/recipe/diff.go`, `internal/recipe/diff_test.go`;
- create `internal/recipe/metadata.go`, `internal/recipe/metadata_test.go`;
- create `internal/recipe/aur_rpc.go`, `internal/recipe/aur_rpc_test.go`;
- create fixtures under `testdata/recipes/identity/` and `testdata/recipes/noexec/`;
- create `test/integration/recipe_noexec_test.go`.

**RED:** fixtures and a fake bounded AUR RPC server for authoritative source/namespace/pkgbase, maintainer/source metadata, committed `.SRCINFO` as hostile data, redirects, malformed/oversized/contradictory RPC records, and RPC/repository identity mismatch; SHA-1/SHA-256 Git object formats; commit/tree/blob identity; raw path-byte preservation; file type/mode/size/hash/blob OID; symlinks, special files, hostile filenames, control/bidi/invalid UTF-8, relevant untracked files, changed modes, workspace device/inode, tree/manifest drift; first-install empty baseline and later diff; marker PKGBUILDs/sources proving no command, makepkg, sourcing, or package content executes.

**GREEN:** query the exact supported AUR RPC contract with a bounded standard-library HTTP client, acquire the authoritative AUR Git candidate with argv-only Git commands, cross-check RPC and repository identity, enumerate immutable tree objects, open workspaces without following sensitive symlink components, compute canonical bounded manifests and diffs, and persist identity/evidence. `.SRCINFO` and AUR RPC data are metadata only; never regenerate `.SRCINFO`.

**REFACTOR:** separate byte identity from escaped display text; make manifest serialization versioned and deterministic.

**Verification:** focused unit/fuzz/integration marker tests; run under `strace` or equivalent in disposable CI to prove forbidden executables are not invoked during collection; cumulative baseline.

**Exit criteria:** identical inputs produce identical identity; mutation of every bound component changes or invalidates it; collection executes no package content. Covers ADR-0005, ADR-0010, ADR-0012, and security gate 2 of ADR-0013.

### 07 — Deterministic scanner and bounded context construction

**Prerequisites:** slice 06 and store core from slice 05.

**Files:**

- create `internal/scanner/finding.go`, `internal/scanner/finding_test.go`;
- create `internal/scanner/rules.go`, `internal/scanner/rules_test.go`;
- create `internal/scanner/context.go`, `internal/scanner/context_test.go`, `internal/scanner/context_fuzz_test.go`;
- create `internal/scanner/benchmark_test.go`;
- populate `testdata/recipes/corpus/benign/` and `testdata/recipes/corpus/suspicious/`;
- create `testdata/recipes/corpus/manifest.json` mapping every advertised indicator to positive and false-positive fixtures.

**RED:** stable finding IDs independent of prose; immutable evidence/severity; line/byte references; aggregate signal separated from findings; positive, benign, false-positive, malformed, obfuscated, prompt-injection, oversized, binary, symlink, control/bidi, and truncation cases for every accepted rule family; exact initial ceilings of 256 KiB/file, 2 MiB aggregate, 512 KiB diff, 200 files; relevant omission yielding `partial` and `unknown|caution`, never `clear`.

**GREEN:** implement bounded lexical/non-executing rules and deterministic context selection. Do not add a shell parser unless a failing rule requirement justifies a separate dependency/ADR review.

**REFACTOR:** version rule and context contracts; keep observations factual and intent-neutral.

**Verification:** `go test ./internal/scanner`; fuzz seed regression; corpus coverage script verifies no advertised indicator lacks positive and benign controls; benchmark the accepted context ceilings against the full corpus and retain results as a release artifact; rerun no-execution marker suite; cumulative baseline.

**Exit criteria:** corpus covers every advertised v1 indicator; scanner output is reproducible and attributable; truncation/failure is visible. Covers ADR-0011 and security gates 1–2 of ADR-0013.

### 08 — Advisory LLM contracts and backend adapters

**Prerequisites:** slices 02, 03, 05, and 07.

**Files:**

- create `internal/store/migrations/0002_llm_assessments.sql` and migration tests;
- create `internal/llm/types.go`, `internal/llm/types_test.go`, `internal/llm/validate_fuzz_test.go`;
- create `internal/llm/select.go`, `internal/llm/select_test.go`;
- create `internal/llm/http.go`, `internal/llm/http_test.go`;
- create `internal/llm/codex.go`, `internal/llm/codex_test.go`;
- create `internal/llm/request.go`, `internal/llm/request_test.go`;
- create fixtures under `testdata/llm/`;
- create `test/contract/codex_contract_test.go` for an explicitly pinned supported CLI version.

**RED:** no decision/action fields; unknown fields/enums/path/line references rejected; UTF-8/string/count/response bounds; package prompt injection remains data; deterministic evidence cannot be replaced; host paths/environment/secrets omitted; explicit backend precedence; automatic complete-API-before-Codex; disabled mode; Codex model default exactly `5.6-luna`; explicit incomplete config, unauthenticated Codex, timeout, refusal, invalid output, and transport failure produce visible `unknown` with no silent backend/model fallback; raw retention and privacy mode behavior.

**GREEN:** implement typed JSON with `DisallowUnknownFields` and semantic validation, standard-library HTTP transport, and argv-only Codex invocation from a private non-recipe directory containing bounded redacted input. Add persistence only with this feature.

**REFACTOR:** share response validation between HTTP and Codex while keeping transport diagnostics distinct; never add an LLM SDK.

**Verification:** `go test ./internal/llm ./internal/store`; mock HTTP tests; fake-Codex tests; opt-in pinned live Codex contract in isolated CI with no package repository cwd; secret-leak canary scan of requests/logs; cumulative baseline.

**Exit criteria:** model output is optional, advisory, locally validated, and incapable of authorizing work; failure remains human-visible. Covers ADR-0008, ADR-0011, and security gate 3 of ADR-0013.

### 09 — Reports, terminal review, and append-only human decisions

**Prerequisites:** slices 02, 05, 07, and 08.

**Files:**

- create `internal/report/render.go`, `internal/report/render_test.go`;
- create `internal/report/escape.go`, `internal/report/escape_fuzz_test.go`;
- create `internal/report/write.go`, `internal/report/write_test.go`;
- create `internal/review/present.go`, `internal/review/present_test.go`;
- create `internal/review/decision.go`, `internal/review/decision_test.go`;
- create `internal/store/migrations/0003_human_decisions.sql` and migration tests;
- create `internal/store/decision.go`, `internal/store/decision_test.go`;
- create golden fixtures under `internal/report/testdata/` and transcript fixtures under `internal/review/testdata/`.

**RED:** text/Markdown/JSON separation of technical status, deterministic findings, aggregate signal, LLM assessment, and null/explicit human decision; provenance and partial/failed limitations; escaped hostile terminal content; atomic private write (`fsync`, rename, directory sync where supported); viewer receives a read-only generated report; terminal menu supports inspect/approve/defer/reject/cancel; a `clear` recipe may use concise/batched presentation but still records explicit human approval before any approval is armed; Paru's later native confirmation is not removed; approval after visible partial/failed analysis remains explicit; edited recipe invalidates identity and requires reinspection; no automatic decision field or path.

**GREEN:** render only from typed store state plus immutable blobs; implement generic viewer/pager launch and append-only decision recording. Reports remain snapshots and are never read as state.

**REFACTOR:** keep rendering pure where possible and presentation I/O behind narrow testable boundaries.

**Verification:** golden tests, transcript tests, terminal escape fuzzing, marker/no-execution suite, cumulative baseline.

**Exit criteria:** a user can understand evidence, provenance, uncertainty, and required action without any signal masquerading as a decision. Covers ADR-0009, ADR-0011, ADR-0012, and security gate 3 of ADR-0013.

### 10 — Transaction planning, drift comparison, and held closure

**Prerequisites:** slices 01, 04, 05, 06, and 09.

**Files:**

- create `internal/paru/plan.go`, `internal/paru/plan_test.go`;
- create `internal/paru/drift.go`, `internal/paru/drift_test.go`;
- create `internal/paru/closure.go`, `internal/paru/closure_test.go`, `internal/paru/closure_fuzz_test.go`;
- create `internal/store/migrations/0004_transactions.sql` and migration tests;
- create `internal/store/transaction.go`, `internal/store/transaction_test.go`;
- create `test/integration/planning_flow_test.go`.

**RED:** explicit target/search/AUR upgrade plans; authoritative repo/AUR origin records; complete split-package/package-base grouping; deterministic argument recording; dependency/provider/version/origin/recipe identity drift; unknown/MISSING/conflicting/partial split-package records; configured PKGBUILD mode neutralization; held pkgbase plus impossible dependant closure; changed re-resolution always returns to review rather than repair; an executable test exposes the non-atomic interval between the last plan comparison and Paru's fresh resolver run; no SQLite transaction held while waiting on a child or user.

**GREEN:** persist planning evidence, derive review candidates and held closure from proven Paru records, and compare a fresh execution-resolution plan with the approved plan. Paru remains authoritative; AURoscope only validates and compares its typed output. Treat this comparison as pre-execution evidence, not an atomic lock on Paru's later resolver state.

**REFACTOR:** keep planning model independent of human presentation and executable orchestration.

**Verification:** focused unit/fuzz tests; synthetic integration with fake runner and SQLite; supported-version Arch contract fixtures; cumulative baseline.

**Exit criteria:** every executable AUR package base is planned and reviewable; any material observed drift fails closed with an explainable difference. Before slice 12, the supported dependency contract must prove how the actual execution run is bounded by the approved closure; if it cannot, open a `MODE: DECISION` amendment and stop rather than overstate the guarantee. Covers ADR-0003, ADR-0004, ADR-0009, and ADR-0014.

### 11 — One-shot approval lifecycle and guard

**Prerequisites:** slices 03, 05, 06, 09, and 10.

**Files:**

- create `internal/approval/identity.go`, `internal/approval/identity_test.go`;
- create `internal/approval/lifecycle.go`, `internal/approval/lifecycle_test.go`;
- create `internal/approval/process_linux.go`, `internal/approval/process_linux_test.go`;
- create `internal/approval/guard.go`, `internal/approval/guard_test.go`;
- create `internal/store/migrations/0005_approvals.sql` and migration tests;
- create `internal/store/approval.go`, `internal/store/approval_test.go`;
- modify `internal/cli/command.go` and `cmd/auroscope/main.go` to wire the internal `guard` command;
- create `test/integration/approval_concurrency_test.go`, `test/integration/guard_identity_test.go`.

**RED:** arm only from same-inspection explicit human approve; 30-minute default and 2-hour maximum; lowercase-hex transaction ID; exact source/namespace/pkgbase, commit/tree/manifest/per-file/scanner/human/workspace/UID/process binding; parent PID/start ticks/executable; pre-claim no-follow identity; `BEGIN IMMEDIATE` atomic single `armed → claimed`; concurrent duplicate claim; expiry; unexpected pkgbase; every identity component mutation; post-claim recomputation; monotonic `claimed → consumed`; terminal invalidation; startup orphan recovery; process evidence unavailable/contradictory; approval after recorded partial/failed analysis.

**GREEN:** implement the store transitions and narrow `guard --transaction <id>` path. The hook string is fixed `exec /usr/bin/auroscope guard --transaction <id>` with no package-derived text. Guard success means final identity verification before any recipe code, not safety or per-build adjacency.

**REFACTOR:** keep process identity Linux-specific and keep manifest implementation shared with slice 06.

**Verification:** race/concurrency integration; crash-injection around claim/verification; mutation matrix; cumulative baseline.

**Exit criteria:** no approval is floating or reusable; missing, duplicate, expired, drifted, substituted, or replayed state fails before recipe-supplied code executes. Covers ADR-0005, ADR-0009, and ADR-0010.

### 12 — Private Paru execution handoff and artifact reuse

**Prerequisites:** slices 02–06 and 10–11.

**Files:**

- create `internal/paru/config.go`, `internal/paru/config_test.go`;
- create `internal/paru/execute.go`, `internal/paru/execute_test.go`;
- create `internal/store/migrations/0006_build_outcomes.sql` and migration tests;
- create `internal/store/build.go`, `internal/store/build_test.go`;
- create `internal/approval/artifact.go`, `internal/approval/artifact_test.go`;
- create `test/contract/paru_config_contract_test.go`;
- create `test/integration/execution_flow_test.go`.

**RED:** original config resolution (`PARU_CONF`, XDG, `/etc/paru.conf`); absolute trusted include; nested includes, missing files, repeated `[bin]`, whitespace paths, and later guard override; private `0600` transaction config; fixed hook; `PARU_CONF` handoff; mandatory `--skipreview`; exact final trusted modes; no added `--noconfirm`; fresh plan comparison before execution; process-session recording before guard; success/child failure/signal/cancel invalidation; force rebuild by default; exact post-build evidence for every split-package artifact before recording a reusable SHA-256; substituted cache file, same name/version from another build, partial outputs, interrupted build, and stale records; unexpected guard call abort before repo build dependencies.

**GREEN:** generate private Paru config, start the execution process group, bind the session, consume/invalidate approvals, and record only outcomes proven by supported process/dependency evidence. Never invoke `makepkg --packagelist`, `--printsrcinfo`, or `--verifysource` to discover artifacts. If no exact bounded post-build surface exists for a case, record the limited outcome and keep artifact reuse disabled; no arbitrary package artifact inherits recipe approval.

**REFACTOR:** separate command construction, config rendering, lifecycle transitions, and child observation.

**Verification:** fake runner integration; exact Paru config parser contract in disposable Arch; artifact mutation/reuse matrix; signal suite; cumulative baseline.

**Exit criteria:** execution uses the reviewed plan and guard, cannot reopen Paru review after the guard, and never reuses an unproven package artifact. Actual execution-closure observation satisfies slice 10's gate or implementation remains blocked for a design amendment. Covers ADR-0003, ADR-0005, ADR-0006, ADR-0010, and ADR-0012.

### 13 — Complete official-upgrade-first orchestration

**Prerequisites:** slices 03–04 and 10–12.

**Files:**

- create `internal/paru/upgrade.go`, `internal/paru/upgrade_test.go`;
- create `internal/paru/orchestrate.go`, `internal/paru/orchestrate_test.go`;
- modify `internal/cli/command.go` and `cmd/auroscope/main.go` for intercepted user flows;
- create `test/integration/upgrade_flow_test.go`.

**RED:** bare invocation and `-Syu`; complete official `paru -Syu --repo` (or proven equivalent) first with inherited terminal and exact status; no AUR review around official packages; failure/cancellation prevents AUR phase; post-upgrade AUR query/plan/review only; explicit install/search flows skip unrelated sysupgrade; deferred/rejected AUR closure passed as exact ignores; changed closure or plan returns to review; transparent flows preserve argv/status.

**GREEN:** implement the top-level state machine using existing classifier, process, adapter, store, review, approval, and execution packages. Keep phase results separate and visible.

**REFACTOR:** model states and terminal outcomes explicitly; do not hide phases in a broad “install” function.

**Verification:** deterministic fake-runner scenarios; supported-version disposable Arch integration with synthetic repositories/private Pacman state; cancellation and child-status matrix; cumulative baseline.

**Exit criteria:** official upgrade is never fragmented for AUR policy; no AUR recipe code runs before post-upgrade review/approval; explicit and search installs use the same exact-review boundary. Covers ADR-0002, ADR-0003, and ADR-0004.

### 14 — Status, held, explain, and snapshot exports

**Prerequisites:** slices 05, 09–13.

**Files:**

- create `internal/store/status.go`, `internal/store/status_test.go`;
- create `internal/report/status.go`, `internal/report/status_test.go`;
- extend `internal/cli/command.go`, `internal/cli/command_test.go`;
- create command tests under `cmd/auroscope/main_test.go`.

**RED:** `status`, `status --verbose`, `status --json`, `status --markdown`, `held`, and `explain <package>`; verbose status includes AURoscope build version/commit while transparent `--version` remains Paru's; known/clear/awaiting/deferred/rejected counts; concise reasons and signal sources; dependency consequences; required human action; identity/decision/build/install history; byte-safe display; stable JSON exit/error envelope; reports generated from SQLite and never read back.

**GREEN:** add read-only store queries and renderers, then wire owned CLI subcommands without interfering with Paru passthrough.

**REFACTOR:** share typed view models, not rendered strings, across output formats.

**Verification:** golden text/Markdown/JSON tests; corrupted/missing DB diagnostics; read-only concurrency with active transaction; cumulative baseline.

**Exit criteria:** all persisted product state required by the specification is explainable without consulting mutable report files. Covers ADR-0009, ADR-0011, and ADR-0012.

### 15 — Cleanup, retention, recovery, and maintenance

**Prerequisites:** slices 02, 05, 11–14.

**Files:**

- create `internal/config/retention.go`, `internal/config/retention_test.go`;
- create `internal/store/recover.go`, `internal/store/recover_test.go`;
- create `internal/store/purge.go`, `internal/store/purge_test.go`;
- create `internal/recipe/cleanup_linux.go`, `internal/recipe/cleanup_linux_test.go`;
- extend `internal/cli/command.go` for `cleanup` and explicit maintenance operations;
- create `test/integration/recovery_cleanup_test.go`.

**RED:** success/error/cancel/signal cleanup; descriptor-relative no-follow deletion; only direct children; recorded marker not authority; UID/device/inode/PID/start-time cross-check; active session never removed; 24-hour runtime grace; unrecorded crash residue; defaults of 365/30/7/14 days, 512 MiB LRU, and 3 days; no unlimited/disabled default; live/referenced identity preservation; serialized age+quota cache prune; explicit backup restore and maintenance-only `VACUUM`; idempotent restart recovery.

**GREEN:** implement startup recovery, explicit cleanup/purge, and bounded retention around recorded state. Never call recursive deletion on a path derived solely from package or marker text.

**REFACTOR:** centralize lifecycle terminal cleanup and make purge decisions independently testable from deletion mechanics.

**Verification:** fake-clock tests; symlink/path mutation integration; crash checkpoints; repeated idempotent recovery; cumulative baseline.

**Exit criteria:** temporary work does not accumulate silently; retained evidence stays bounded without deleting live or referenced records. Covers ADR-0009, ADR-0010, and ADR-0012.

### 16 — Full functional integration matrix

**Prerequisites:** slices 00–15.

**Files:**

- create scenario fixtures under `test/integration/scenarios/`;
- create `test/integration/transparent_flow_test.go`, `selection_flow_test.go`, `explicit_install_flow_test.go`, `aur_upgrade_flow_test.go`, `failure_flow_test.go`;
- extend `scripts/check.sh` with deterministic integration targets.

**RED:** add one failing scenario at a time for repo-only pass-through, package archive `-U`, search selection/cancel, explicit repo target, explicit AUR target, mixed repo/AUR dependencies, bare upgrade, AUR-only upgrade, first install, differential update, partial/failed scan, disabled/HTTP/Codex LLM, approve/defer/reject/cancel, plan drift, guard drift, artifact reuse/rebuild, child failure/signal, crash recovery, and status/explain results.

**GREEN:** fix only the integration defect exposed by the current scenario, then run the focused scenario before advancing to the next one.

**REFACTOR:** remove duplicate scenario wiring while preserving typed boundaries; do not replace live contract gates with mocks.

**Verification:** cumulative baseline plus `CGO_ENABLED=1 go test -race ./test/integration/...`; coverage report used to find untested branches, not as a substitute acceptance metric.

**Exit criteria:** every product acceptance criterion has at least one deterministic integration scenario and a traceable test name.

### 17 — Arch packaging, dependency inventory, and release CI

**Prerequisites:** slices 00–16.

**Files:**

- create `packaging/arch/PKGBUILD` and packaging test fixtures;
- create `scripts/build-arch.sh`, `scripts/check-package.sh`, `scripts/generate-sbom.sh`;
- extend `.gitea/workflows/verify.yml` and create `.gitea/workflows/release.yml`;
- update `README.md` with verified installation/support boundaries;
- create `docs/compatibility.md` and `docs/security-model.md` distilled from accepted contracts.

**RED:** packaging smoke fails before required metadata/dependencies/install layout exist; package inspection checks Go/C toolchain are make dependencies, not runtime dependencies; embedded version/commit is visible through `status --verbose` and artifact metadata without stealing Paru's transparent `--version`; release SQLite runtime matches the pinned library version and `PRAGMA compile_options` baseline; no unsupported static/pure-Go claim; checksums/module list/SBOM produced; package installs in disposable root; unsupported Paru/Pacman/makepkg diagnostics are explicit; clean rebuild comparison records reproducibility or documented variance.

**GREEN:** implement self-hosted AUR-style package build for Arch `linux/amd64`, release artifacts, checksums, SBOM/module inventory, and exact compatibility documentation. Do not publish to `aur.archlinux.org`.

**REFACTOR:** keep packaging logic in PKGBUILD/scripts rather than application runtime.

**Verification:** clean Arch builds twice; package install/uninstall smoke in disposable environment; `namcap` where useful; module/license inventory; full baseline and contract matrix.

**Exit criteria:** a user can install the self-hosted package without a Go runtime; release artifacts identify source and dependencies; reproducibility result is honest. Covers ADR-0001, ADR-0007, and ADR-0008.

### 18 — Mandatory disposable-Arch security proof and release candidate gate

**Prerequisites:** slices 00–17 and all required supported-version artifacts.

**Files:**

- create `test/e2e/run-arch.sh`, `test/e2e/guard-host.sh`;
- create `test/e2e/fixtures/` for synthetic Pacman repository, AUR Git repositories, malicious markers, and scripted user input;
- create `test/e2e/e2e_test.go` with machine-verifiable assertions;
- create `docs/release-checklist.md`;
- extend release CI to archive logs, reports, DB snapshot, and package hashes from the disposable environment.

**RED:** the harness must first prove its safety guard rejects real `/`, host `/var/lib/pacman`, host cache, absent private database/root, privileged host mounts, and non-disposable execution. Then add failing end-to-end assertions before wiring each scenario.

**GREEN:** in a disposable Arch container or VM, create a synthetic signed/private repository and synthetic AUR remotes, then exercise selection/planning → non-executing collection → deterministic/optional LLM evidence → visible human decision → exact guard → build/install. Include suspicious/benign recipes, marker proof, defer/reject, drift abort, and a successful explicit approval. Never target the workstation.

**REFACTOR:** keep the security proof narrow: hostile package input, no execution during inspection, faithful evidence, human authority, and order-before-install. Ordinary compatibility scenarios may share infrastructure but remain labelled functional tests.

**Verification:** full baseline; supported-version contracts; corpus coverage; package build; complete disposable E2E; manual review of generated report wording to ensure it says “reviewed” rather than “safe”.

**Exit criteria:** all four ADR-0013 gates pass; all specification acceptance criteria pass; no unresolved critical/high release blocker; Mathieu reviews the release checklist before any v1 tag.

## 6. Dependency graph and merge checkpoints

```text
00
├─ 01
├─ 02
│  ├─ 05
│  └─ 03
│     └─ 04
├─ 05 ─┬─ 06 ── 07 ── 08 ── 09
│      └──────────────────────┘
├─ 04 + 05 + 06 + 09 ── 10
├─ 03 + 05 + 06 + 09 + 10 ── 11
├─ 02..06 + 10 + 11 ── 12
├─ 03 + 04 + 10..12 ── 13
├─ 05 + 09..13 ── 14
├─ 02 + 05 + 11..14 ── 15
└─ 00..15 ── 16 ── 17 ── 18
```

Review checkpoints:

- **Foundation checkpoint after 05:** module, CLI boundary, private config/runtime, process behavior, dependency adapters, and store mechanics are individually proven.
- **Inspection checkpoint after 09:** exact identity, no-execution collection, scanner, LLM, reports, and human authority work without package installation.
- **Execution checkpoint after 13:** planning drift, one-shot guard, artifact binding, and official-first orchestration work in disposable integration.
- **Operations checkpoint after 16:** status, recovery, cleanup, retention, and the complete deterministic functional matrix pass.
- **Release checkpoint after 18:** packaging, exact supported dependency contracts, and mandatory disposable-Arch proof pass.

Later slices may be specified while an earlier PR is under review, but their implementation branches must start from a base containing every prerequisite.

## 7. Requirement traceability

| Accepted decision / product criterion | Primary slices |
| --- | --- |
| ADR-0001 — Go and self-hosted Arch packaging | 00, 17 |
| ADR-0002 — Paru native selection | 04, 13, 16, 18 |
| ADR-0003 — multi-stage Paru orchestration | 04, 10, 12, 13 |
| ADR-0004 — official upgrade before AUR review | 10, 13, 16, 18 |
| ADR-0005 — final recipe identity guard | 06, 11, 12, 18 |
| ADR-0006 — Go/process architecture | 01, 03, 04, 12 |
| ADR-0007 — `mattn/go-sqlite3` with CGO | 00, 05, 17 |
| ADR-0008 — minimal direct dependencies | 02, 05, 08, 17 |
| ADR-0009 — instrumental SQLite state | 05, 10, 11, 14, 15 |
| ADR-0010 — one-shot approvals | 06, 11, 12, 15, 18 |
| ADR-0011 — deterministic scanner/advisory LLM | 02, 07, 08, 09, 14 |
| ADR-0012 — XDG/private cleanup/retention | 02, 05, 06, 09, 12, 15 |
| ADR-0013 — bounded AUR threat model and four v1 gates | 06, 07, 09, 18 |
| ADR-0014 — reject local PKGBUILD inputs | 01, 04, 10, 16 |
| Bare `auroscope` replacement | 13, 16, 18 |
| Native numbered selection | 04, 13, 18 |
| Official packages without AUR review friction | 01, 03, 13, 18 |
| AUR recipes/dependencies inspected before build | 06–13, 18 |
| Deterministic and LLM evidence remain separate | 07–09, 14, 18 |
| User owns every consequential decision | 09, 11, 13, 18 |
| `status`, `held`, `explain` expose held consequences | 10, 14, 16 |
| Approval binds exact commit and complete manifest | 06, 11, 18 |
| Changed recipe stops before recipe code | 11, 12, 18 |
| Cancellation leaves no reusable approval | 03, 11–13, 16 |
| Disposable E2E never alters workstation | 04, 13, 17, 18 |

## 8. Definition of v1 implementation complete

Implementation is complete only when:

- every slice is merged through its own authorized execution issue/PR and all prerequisites are traceable;
- cumulative unit, race, vet, formatting, fuzz-seed, integration, supported-version contract, packaging, and disposable E2E gates pass;
- the recipe corpus covers every advertised deterministic indicator with suspicious and benign/false-positive controls;
- marker tests prove collection, scanning, context preparation, and presentation execute no package content;
- generated reports visibly separate deterministic evidence, LLM advice, technical completeness, and human decision;
- exact plan/recipe/process/workspace approval binding and artifact reuse rules survive concurrency, drift, cancellation, signal, and recovery tests;
- bare upgrade runs a complete official phase first and never starts AUR review after official failure/cancellation;
- local PKGBUILD inputs fail before Paru starts;
- status and cleanup behavior are observable and finite by default;
- the self-hosted Arch package installs in a disposable environment, identifies its source version, and does not require Go at runtime;
- the release checklist documents remaining compatibility limits and residual risks without calling reviewed packages safe;
- Mathieu reviews the release-candidate evidence and separately authorizes tagging/release.

## 9. Explicitly deferred beyond v1

Do not let implementation slices absorb these items:

- publication to `aur.archlinux.org`;
- a replacement TUI or editor plugin;
- a Paru/libalpm resolver implementation;
- local PKGBUILD or configured PKGBUILD-repository support;
- exhaustive upstream-source or compiled-binary analysis;
- custom build sandbox/chroot guarantees or hostile-local-account protection;
- generic plugin/rule frameworks, event sourcing, provenance/EAV schemas, or distributed orchestration;
- autonomous non-interactive approve/reject policy;
- cross-platform or pure-Go SQLite release targets;
- PTY support without a failing executable contract test against a required supported dependency.
