# AURoscope v1 implementation plan

**Status:** Definitive implementation plan, pending acceptance and merge of [PR #18](https://git.2027a.net/2027a/auroscope/pulls/18). This document is executable guidance, not authorization to write production code. Implementation starts only after this plan is merged and Mathieu explicitly starts the execution loop.

**Design baseline:** [`README.md`](../../README.md), [`docs/specification.md`](../specification.md), [`docs/design-phase.md`](../design-phase.md), and accepted [ADR-0001](../decisions/0001-go-and-self-hosted-arch-packaging.md) through [ADR-0014](../decisions/0014-reject-local-pkgbuild-inputs-in-v1.md).

## 1. Mission

Build a dependable personal Arch Linux tool, not a demonstration and not a general package-management platform.

AURoscope v1 must be useful as Mathieu's normal Paru entry point for the supported surface. It must inspect exact AUR recipes before recipe-supplied code executes, present deterministic and optional LLM evidence without making a decision, bind explicit human approval to the reviewed identity, and delegate selection, dependency resolution, builds, and installation to Paru/Pacman.

The project is personal software:

- correctness at the recipe-review and approval boundary matters;
- ordinary failures must be diagnosable and recoverable;
- the implementation must remain small enough for one maintainer to understand;
- abstractions, compatibility layers, test suites, and release machinery need a demonstrated v1 use;
- there is no requirement to build a generic framework, enterprise release process, or hostile-local-account security boundary.

## 2. Non-negotiable product boundaries

Every milestone preserves these accepted constraints:

- Paru remains selector, resolver, builder, and Pacman frontend. AURoscope does not implement another resolver.
- Official repository upgrades run first as one complete native phase. AUR review begins only after that phase succeeds.
- Inspection never executes or sources `PKGBUILD`, package source material, or another recipe-supplied file.
- Package content, metadata, paths, diffs, fixtures, and model input are hostile data, never instructions.
- Technical status, deterministic findings, aggregate signal, LLM assessment, and human decision remain separate.
- Only an explicit human `approve` decision may arm an approval. A clear signal is not an approval.
- Approval binds the exact recipe identity accepted in ADR-0010 and is one-shot.
- The final guard verifies the complete identity before recipe-supplied code executes and execution uses `--skipreview`.
- Local/path PKGBUILD inputs and configured PKGBUILD-repository records remain unsupported and fail closed in v1.
- Automated package-manager tests never touch the real workstation root, Pacman database, or package cache.
- AURoscope says a recipe was reviewed; it never says a package is safe.

If executable evidence contradicts an accepted ADR, implementation stops at that boundary. The agent opens a separate `MODE: DECISION` issue with the evidence and does not improvise a new architecture in code.

## 3. Deliberate simplifications

These choices keep v1 substantial but proportionate.

### 3.1 Vertical delivery before subsystem breadth

The first real implementation path is one explicit AUR target through planning, collection, deterministic review, explicit approval, guard, rebuild, and execution in disposable Arch. Search, bare upgrades, the full rule catalogue, LLM backends, retention, and packaging expand that working path later.

### 3.2 Prove Paru execution feasibility first

Before durable application architecture grows around it, an executable disposable-Arch spike must prove:

1. native planning and the second resolution run;
2. the observable closure used by the actual execution invocation;
3. the private `PARU_CONF` handoff;
4. `PreBuildCommand` invocation before recipe code;
5. `--skipreview` behavior;
6. fail-closed detection of an unplanned or changed package base.

If the actual execution closure cannot be bounded by a supported Paru surface, the loop stops after the spike and requests a design amendment. It does not build SQLite, scanner, LLM, or review layers around an unproved assumption.

### 3.3 No package-artifact reuse in v1

V1 always forces AUR rebuilds. A cached or previously built package is never reused because a recipe was approved.

Build and installation outcomes are recorded only to the extent supported by bounded evidence needed by `status` or `explain`. Exact artifact hashes may be recorded when a supported surface provides them, but they do not authorize reuse. A reusable-artifact feature belongs to a later decision and implementation slice.

### 3.4 LLM is additive, not on the critical path

The deterministic workflow and human decision path work completely with LLM mode `disabled`. HTTP and Codex adapters are added only after guarded execution works end to end. LLM failure remains visible evidence about assessment completeness, never a blocker imposed by architecture and never an autonomous decision.

### 3.5 Configuration and schema grow with consumers

Strict TOML, forward SQL migrations, and accepted XDG roots remain. Configuration keys, tables, columns, and operational machinery appear only in the milestone that consumes them. No up-front skeleton is created for future scanner, LLM, retention, compatibility, or release features.

### 3.6 Proportionate verification

- Pure logic receives focused unit/table/fuzz-seed tests.
- External Paru/Pacman/makepkg behavior receives one supported-version contract suite.
- Cross-package behavior uses one integration harness grown incrementally.
- ADR-0013 receives one disposable-Arch E2E path plus critical abort cases.
- Targeted race tests run where concurrency exists. The full race suite runs at milestone checkpoints and release, not after every small edit.
- TDD is the implementation method; intentionally red commits and repeated failure transcripts are not deliverables.
- Generated reports are private, closed, and atomically renamed. SQLite and migrations are the durability boundary; reconstructible reports do not need database-grade directory synchronization.

## 4. Delivery model: one autonomous implementation loop

### 4.1 Durable sources of truth

The loop relies only on durable state a fresh session can read:

1. this plan and the accepted ADRs;
2. one Forgejo umbrella issue whose first non-empty line is `MODE: EXECUTION` and which records Mathieu's explicit advance from design to implementation; the marker alone is not a phase transition;
3. one long-lived branch, `implementation/v1`;
4. one draft PR from `implementation/v1` to `main`;
5. `docs/implementation/v1-status.md` on that branch;
6. commits, test output summaries, and Forgejo comments.

Chat history is never required to resume.

### 4.2 Why one branch and one PR

The seven milestones below are implementation checkpoints, not seven mandatory merge queues and not dozens of subsystem PRs. The default autonomous path uses one branch and one draft PR so the loop can continue without asking Mathieu to merge intermediate scaffolding.

Each milestone ends in one or more coherent green commits and a concise PR progress comment. The agent never merges the final PR. Mathieu reviews the complete branch or may request an intermediate review at any checkpoint.

### 4.3 Preferred Hermes execution mode

On Mathieu's Hermes host, bootstrap the work as a serial Kanban chain:

- one orchestrator task owns the umbrella issue, branch, draft PR, and status file;
- one worker task is created for each milestone below;
- task dependencies enforce milestone order; only one implementation task may be ready or claimed at a time, and implementation workers never run concurrently against `implementation/v1` or its worktree;
- every task carries the exact repository path, branch, umbrella issue, draft PR, plan path, and current status-file path so a worker does not infer them from chat;
- each worker reads the repository and live status, completes or resumes exactly one milestone, verifies it, commits, pushes, updates status, and completes or blocks its task;
- the dispatcher advances the next ready task automatically;
- the orchestrator verifies milestone evidence and only intervenes to repair routing, record a blocker, or finalize the release-candidate handoff. It does not duplicate worker implementation.

A persistent `/goal` session may execute the same state machine directly when Kanban is unavailable. The repository status file and Forgejo objects remain authoritative in either mode.

### 4.4 Loop state file

`docs/implementation/v1-status.md` is created by the bootstrap session on `implementation/v1`. Keep it short and overwrite current state rather than accumulating a journal.

It records:

```markdown
# AURoscope v1 execution status

- umbrella issue: <URL>
- draft PR: <URL>
- branch: implementation/v1
- current milestone: <number and name>
- state: pending | active | blocked | complete
- last verified commit: <SHA>
- last checks: <commands and results>
- next action: <one concrete action>
- blocker/decision issue: <none or URL>
```

Forgejo comments provide the history; this file provides the restart point.

### 4.5 Worker cycle

For each milestone, the worker must:

1. verify repository, branch, identity, worktree, umbrella issue, draft PR, and current status;
2. read this plan, relevant ADRs, existing code/tests, and the previous milestone report;
3. confirm the milestone Definition of Ready;
4. split the milestone internally into small RED → GREEN → REFACTOR cycles;
5. implement the smallest vertical behavior that reaches the milestone Definition of Done;
6. run focused checks while editing, then the milestone verification matrix;
7. review the diff for scope, unsafe package-content handling, hidden design changes, and unnecessary abstraction;
8. update `v1-status.md`;
9. create signed, coherent, green commits and push `implementation/v1` without force;
10. update the draft PR with the milestone result and immediately advance the next ready milestone.

The loop does not stop merely because one task, test, or commit completed. If a worker reaches its session or tool budget before the milestone DoD, it commits and pushes only a coherent green checkpoint, leaves the milestone task incomplete with a resume comment, and the orchestrator requeues that same milestone from `v1-status.md`; the next milestone must not become ready.

### 4.6 Autonomous continue and stop policy

Continue without asking Mathieu when:

- the next action is inside an accepted milestone;
- failures are ordinary implementation defects with a safe local fix;
- tests or supported-version contracts reveal missing code but not a design contradiction;
- a refactor is local and reduces duplication without changing public behavior;
- a session limit requires a clean checkpoint and resumption from `v1-status.md`.

Stop and report when:

- Milestone 0 cannot prove the execution-closure contract;
- evidence contradicts an accepted ADR or requires a consequential product choice;
- a new direct dependency, PTY, resolver behavior, local PKGBUILD support, artifact reuse, or threat-model expansion appears necessary;
- a test would risk the real workstation package state;
- credentials or external infrastructure required by a mandatory gate are unavailable after safe alternatives are exhausted;
- the same blocker survives two bounded implementation attempts;
- the release candidate is ready for Mathieu's review and tag authorization.

When blocked, preserve a coherent branch, update `v1-status.md`, push the checkpoint, comment the evidence on the umbrella issue/PR, and create a focused `MODE: DECISION` issue only when a real accepted-design change is needed.

## 5. Repository shape

Create packages only when the working vertical path needs them. The accepted ADR-0006 domain boundaries remain the intended final shape:

```text
cmd/auroscope
internal/cli
internal/config
internal/process
internal/paru
internal/recipe
internal/scanner
internal/llm
internal/review
internal/store
internal/approval
internal/report
```

These are domain packages, not interface requirements. Interfaces exist only at nondeterministic boundaries tests must replace: clock/randomness, process execution, model transport, and short store transactions.

This plan intentionally names stable package roots and milestone artifacts rather than preallocating dozens of source files. Each worker chooses the smallest concrete files required by the next failing test and records the resulting paths in the milestone report.

Tests live beside their package by default. Use shared `test/contract`, `test/integration`, `test/e2e`, and `testdata` only when a real process or dependency boundary justifies them.

## 6. Common verification commands

After bootstrap, the ordinary local baseline is:

```console
CGO_ENABLED=1 go test ./...
go vet ./...
test -z "$(git ls-files -z '*.go' | xargs -0 -r gofmt -l)"
git diff --check
```

Use targeted race checks while implementing process, store, and approval concurrency. At milestones 2, 3, 5, and 6 run:

```console
CGO_ENABLED=1 go test -race ./...
```

Live Paru/Pacman/makepkg and installation tests run only through the disposable-Arch harness with explicit private root, database, and cache. The harness must abort before starting if it could target `/`, `/var/lib/pacman`, or the host package cache.

Network/live-provider checks remain opt-in unless they are part of the supported-version release gate. Deterministic tests must not require the network.

## 7. Milestones

### Milestone 0 — Bootstrap and prove the Paru execution contract

**Objective:** establish a minimal Go/CI skeleton and prove the highest-risk external contract before durable architecture grows around it.

**Definition of Ready:**

- this plan is merged;
- Mathieu explicitly starts the implementation loop and that authorization is recorded in the umbrella issue;
- the umbrella `MODE: EXECUTION` issue, `implementation/v1` branch, draft PR, and `v1-status.md` exist;
- the branch starts from current `origin/main`.

**Implement:**

- first update the current-phase statements in `AGENTS.md`, `README.md`, and `docs/design-phase.md` to record the authorized implementation phase without rewriting accepted design history;
- minimal `go.mod` and wiring-only `cmd/auroscope/main.go`;
- a small `scripts/check-docs.sh` that validates repository-relative links and stale phase statements without introducing a documentation framework;
- raw argv classification sufficient for pass-through, one explicit intercepted `-S` case, `--noconfirm` rejection when a new AUR decision is possible, and the ADR-0014 v1 rejection boundary (`-B`, targetless `-U`, local/path-like targets, `file:`, and `pkgbuilds` mode);
- minimal argv-only process runner preserving terminal descriptors and child exit/signal evidence;
- version/capability checks for the exact supported Paru/Pacman/makepkg floor;
- disposable-Arch contract harness;
- in-memory planning/approval placeholders only where needed to exercise the sequence;
- private Paru config and fixed guard probe sufficient to observe the real execution path.

**Do not implement:** SQLite, full configuration, scanner catalogue, LLM, status commands, retention, packaging, or reusable artifacts.

**Verification:**

- documentation checks pass after the phase transition;
- byte-for-byte pass-through fixtures;
- fail-closed local/path input fixtures;
- real process-group cancellation smoke;
- disposable-Arch proof of plan → second resolution → execution closure → hook → `--skipreview`;
- changed or unexpected package-base abort before recipe code;
- host-package-state safety guard.

**Definition of Done:**

- the supported Paru execution closure and hook boundary are executable facts;
- unsupported grammar fails closed;
- if proof fails, a decision issue contains exact commands/output and later milestones remain blocked;
- if proof succeeds, its fixtures become the single supported-version contract harness used later.

### Milestone 1 — First deterministic guarded vertical slice

**Objective:** make one explicit AUR target complete the real product path in disposable Arch with LLM disabled.

**Definition of Ready:** Milestone 0 is complete and the execution contract needs no design amendment.

**Implement:**

- only the XDG paths, private permissions, viewer fallback, and configuration fields consumed by this milestone;
- pinned `go-sqlite3` with the supported SQLite library version, build-tag policy, and `PRAGMA compile_options` baseline recorded and tested;
- minimal forward migrations and SQLite rows for recipe identity, inspection, explicit decision, transaction, approval, process session, build, package-artifact, and installation outcome evidence required by ADR-0009; artifact rows are populated only from bounded exact evidence and never feed reuse;
- a bounded standard-library AUR RPC client for authoritative source/namespace/pkgbase and maintainer/source metadata, tested against a fake server for redirects, malformed/oversized/contradictory records, and RPC/repository mismatch;
- exact AUR Git acquisition, RPC/repository identity cross-check, and canonical tracked recipe manifest without sourcing or executing package content; committed `.SRCINFO` remains hostile metadata and is never regenerated;
- first-install diff/context;
- one representative deterministic rule plus technical status and advisory signal plumbing;
- private text report and terminal `approve | inspect | defer | reject | cancel` decision;
- one-shot approval, complete guard identity verification, private execution handoff, and forced rebuild;
- one cross-component integration harness reused by later milestones.

Treat Git object IDs as validated opaque identities for the supported AUR contract. Do not build a speculative multi-format Git framework.

**Verification:**

- supported SQLite version and compile-option checks;
- empty and reopen migration tests;
- marker recipe proving collection/review executes no package content;
- exact identity mutation rejection;
- explicit decision required even for a clear signal;
- cancellation leaves no reusable approval;
- disposable-Arch explicit install succeeds only after visible evidence and approval;
- reports remain non-authoritative SQLite snapshots.

**Definition of Done:** one explicit AUR package can be planned, reviewed, explicitly approved, guarded, rebuilt, and installed in disposable Arch without LLM or host package-state access.

### Milestone 2 — Complete the inspection and approval trust core

**Objective:** broaden the first vertical slice until its inspection and one-shot approval contracts satisfy the accepted v1 security/correctness boundary.

**Definition of Ready:** Milestone 1's vertical path is green and remains the integration harness.

**Implement:**

- complete canonical manifest handling required by ADR-0010, including tracked file types, modes, paths, hashes, workspace binding, and visible malformed/oversized cases;
- differential identity and recipe diff;
- the advertised deterministic rule catalogue with bounded context and versioned stable finding IDs;
- benign, suspicious, and false-positive corpus entries for every advertised indicator;
- explicit `complete | partial | failed` status and `clear | informational | caution | high | unknown` signal behavior;
- full append-only human decision and approval lifecycle;
- atomic `armed → claimed → consumed` transitions, expiry, duplicate claim rejection, process/workspace binding, terminal invalidation, and startup orphan handling required by ADR-0010;
- readable evidence/provenance and terminal escaping.

**Verification:**

- corpus coverage check;
- fuzz-seed regression for classifier, manifests, scanner context, and terminal escaping;
- no-execution marker across collection, scanning, context construction, and presentation;
- mutation matrix for every approval-bound component;
- targeted store/approval concurrency and crash checkpoints;
- full race baseline.

**Definition of Done:** all four ADR-0013 proof areas except the final release E2E have concrete reusable fixtures, and no signal or failure can masquerade as a decision.

### Milestone 3 — Complete the Paru-facing daily workflow

**Objective:** expand the guarded vertical path into the supported daily Paru replacement surface.

**Definition of Ready:** Milestone 2's identity, evidence, decision, and approval boundaries are stable.

**Implement:**

- transparent pass-through with original argv and child status;
- Paru-native search/numbered selection using the accepted versioned adapter;
- explicit repo, AUR, and mixed-target handling;
- authoritative origin, package-base, split-package, dependency/provider, and version records;
- fresh-plan drift comparison and held/dependant closure;
- bare invocation and `-Syu` as complete official repository phase first, followed only on success by AUR planning/review/execution;
- `status`, `status --verbose`, `status --json`, `status --markdown`, `held`, and `explain <package>` from SQLite;
- concise diagnostics and stable AURoscope-owned exit categories while preserving child exit/signal evidence.

**Verification:**

- supported-version selection, cancellation, order grammar, and mode-reset contracts;
- pass-through, explicit install, search, mixed dependency, bare upgrade, AUR-only upgrade, defer/reject, drift, failure, and signal scenarios in the shared integration harness;
- official-phase failure/cancellation prevents AUR review;
- status views explain held consequences and required action;
- full race baseline.

**Definition of Done:** the specification's supported Paru-facing workflows operate through one guarded architecture, and every acceptance criterion has a named test or a documented release-gate dependency.

### Milestone 4 — Add optional LLM assessment

**Objective:** add useful model explanation without coupling it to decisions or destabilizing the deterministic path.

**Definition of Ready:** Milestone 3 works with LLM mode `disabled`.

**Implement:**

- only the `[llm]` configuration fields now consumed;
- migration for LLM assessments;
- one shared typed response validator with unknown-field, enum, size, path, and line-reference checks;
- standard-library OpenAI-compatible HTTP adapter;
- Codex CLI adapter running outside the recipe repository with bounded redacted input;
- accepted explicit and automatic backend selection, exact default model, timeout/failure behavior, privacy mode, and bounded retention metadata;
- report integration that keeps deterministic evidence immutable and visibly separate.

**Verification:**

- mock HTTP and fake-Codex tests;
- pinned opt-in Codex contract in isolated CI;
- prompt-injection fixtures remain inert data;
- secret/host-path canaries are absent from requests and logs;
- disabled, invalid, timeout, refusal, and unauthenticated states remain visible `unknown` and preserve explicit human authority;
- deterministic end-to-end workflow remains green with LLM disabled.

**Definition of Done:** both supported LLM transports are strictly advisory and locally validated, while the product remains fully operable without them.

### Milestone 5 — Personal-software reliability and bounded maintenance

**Objective:** make normal long-term use boring: private state, finite growth, clear recovery, and no silent temporary-work accumulation.

**Definition of Ready:** real lifecycle rows and configuration consumers exist from milestones 1–4.

**Implement:**

- remaining accepted retention configuration and defaults;
- pre-migration backup/restore when a real prior schema exists;
- startup `quick_check`, stale approval/session recovery, and concise diagnostics;
- descriptor-relative cleanup required by ADR-0012 for AURoscope-owned direct children;
- `auroscope cleanup`, bounded report/model/backups/history retention, and recipe-cache age/quota pruning;
- read-only status while one mutating wrapper is active;
- maintenance-only `VACUUM` if retained as useful.

Do not add a generic scheduler, daemon, event log, or local-host hardening suite. Cleanup and recovery protect ordinary correctness under the accepted trusted-local-account model.

**Verification:**

- fake-clock retention tests;
- success/error/cancel/signal cleanup;
- active session preservation and stale 24-hour recovery;
- symlink/path mutation cases originating from hostile recipe content;
- migration backup/restore from each schema that actually shipped on the branch;
- idempotent restart recovery;
- full race baseline.

**Definition of Done:** default state growth is finite, recoverable, visible, and safe for personal daily use without background services.

### Milestone 6 — Packaging and release-candidate proof

**Objective:** produce an installable self-hosted Arch package and the bounded evidence Mathieu needs to approve v1.

**Definition of Ready:** milestones 0–5 are complete with no unresolved design blocker.

**Implement:**

- self-hosted AUR-style `packaging/arch/PKGBUILD` for Arch `linux/amd64`;
- `-trimpath` build with source version/commit visible through `status --verbose` and Go build metadata;
- Go and C toolchain as build dependencies, not runtime dependencies;
- checksums and standard Go module/license inventory; use ordinary tooling rather than a bespoke SBOM framework;
- exact compatibility and security-boundary documentation;
- minimal release CI and checklist;
- final disposable-Arch E2E using synthetic repository/AUR fixtures.

Do not publish to `aur.archlinux.org`, tag a release, or merge the PR autonomously.

**Verification:**

- clean package build and install/uninstall in disposable Arch;
- packaged SQLite runtime matches the pinned library version and compile-option baseline;
- full unit, vet, formatting, race, contract, integration, corpus, and marker suites;
- final E2E proves planning → non-executing inspection → attributable evidence → explicit decision → exact guard → rebuild/install;
- critical abort cases cover defer/reject, plan drift, guard identity drift, and official-phase failure;
- generated language says “reviewed”, not “safe”;
- `git diff --check` and clean/synchronized implementation branch.

**Definition of Done:** the draft PR is a release candidate with all specification acceptance criteria mapped to passing evidence, remaining compatibility limits documented, and no unresolved critical/high blocker. The loop stops for Mathieu's review and separate merge/tag authorization.

## 8. Checkpoints and requirement traceability

| Boundary | Primary milestone(s) |
|---|---|
| ADR-0001 — Go and self-hosted Arch packaging | 0, 6 |
| ADR-0002 — native Paru selection | 0, 3 |
| ADR-0003 — multi-stage orchestration | 0, 1, 3 |
| ADR-0004 — official upgrade first | 3, 6 |
| ADR-0005 — final recipe guard | 0–2, 6 |
| ADR-0006 — process architecture | 0, 1, 3 |
| ADR-0007 — CGO SQLite | 1, 5, 6 |
| ADR-0008 — minimal dependencies | 1, 4, 6 |
| ADR-0009 — instrumental state | 1–5 |
| ADR-0010 — one-shot approval | 1, 2, 6 |
| ADR-0011 — scanner and advisory LLM | 1, 2, 4 |
| ADR-0012 — XDG, cleanup, retention | 1, 5 |
| ADR-0013 — bounded four-part proof | 1, 2, 6 |
| ADR-0014 — reject local PKGBUILD inputs | 0, 3 |
| Bare `auroscope` replacement | 3, 6 |
| Exact inspected recipe before build | 1, 2, 6 |
| Deterministic/LLM evidence separation | 2, 4, 6 |
| Human owns every decision | 1–4, 6 |
| `status`, `held`, `explain` | 3, 5 |
| Cancellation leaves no approval | 1–3, 6 |
| Workstation package state untouched | 0, 1, 3, 6 |

Review checkpoints are informational by default and do not pause the autonomous loop:

- **Feasibility checkpoint:** Milestone 0.
- **First useful product checkpoint:** Milestone 1.
- **Trust-core checkpoint:** Milestone 2.
- **Daily-use checkpoint:** Milestone 3.
- **Release-candidate checkpoint:** Milestone 6, which always pauses for Mathieu.

Mathieu may ask the loop to pause at any checkpoint.

## 9. Definition of v1 complete

V1 implementation is complete only when:

- Milestone 0 proved the exact supported Paru execution contract before dependent architecture was built;
- bare upgrade, native selection, explicit install, and transparent pass-through satisfy the specification;
- official packages receive no AUR-specific review friction;
- all AUR recipes and required dependencies are inspected before recipe-supplied code;
- deterministic and LLM evidence remain distinct and no automatic decision exists;
- explicit approval binds the exact complete recipe identity and is one-shot;
- drift, cancellation, failure, signal, expiry, or replay leaves no reusable approval;
- v1 always rebuilds AUR packages and never treats recipe approval as cached-artifact approval;
- status, held consequences, decisions, and outcomes are readable from SQLite;
- temporary work and retained state are finite by default and recover correctly;
- the four ADR-0013 gates and specification acceptance criteria pass in disposable environments;
- the self-hosted Arch package installs without a Go runtime;
- the implementation branch and draft PR contain verifiable milestone reports and no unresolved release blocker;
- Mathieu separately approves merge and release/tagging.

## 10. Explicitly deferred

Do not absorb these into the autonomous loop:

- publication to `aur.archlinux.org`;
- cached package-artifact reuse;
- replacement TUI or editor plugins;
- Paru/libalpm resolver implementation;
- local PKGBUILD or configured PKGBUILD-repository support;
- exhaustive upstream-source or compiled-binary analysis;
- custom build sandbox/chroot guarantees or hostile-local-account protection;
- generic plugin/rule frameworks, event sourcing, EAV/provenance systems, daemons, or distributed orchestration;
- autonomous non-interactive package approval or rejection;
- cross-platform or pure-Go SQLite targets;
- PTY support without a failing supported-version contract test;
- bespoke SBOM/reproducibility infrastructure beyond ordinary module inventory, checksums, and honest build metadata.

## 11. Fresh-session bootstrap prompt

After PR #18 is merged, Mathieu can start a new Hephaistos session with this compact prompt:

```text
/goal
Repo: /home/mathieu/.hermes/profiles/hephaistos/projects/auroscope
Target: implement AURoscope v1 through the autonomous loop in docs/implementation/initial-plan.md.
Policy: one MODE: EXECUTION umbrella issue, branch implementation/v1, one draft Forgejo PR; never merge or tag autonomously.

Load first: hermes-agent, kanban-agent-workflows, goal-series-development-loop, software-development-practices, forgejo.
Read AGENTS.md, README.md, docs/specification.md, docs/design-phase.md, every accepted ADR, and the definitive plan.

This prompt authorizes creation of the durable Kanban chain and implementation work described by the plan after verifying PR #18 is merged into current origin/main. Execute milestones serially until the release-candidate gate or a documented stop condition. Prefer direct implementation workers over planning-only output. Persist every restart point in docs/implementation/v1-status.md and Forgejo; do not rely on chat history. Verify, sign, commit, push, and update the draft PR at every milestone. If a design boundary fails, stop with evidence and open the required focused decision issue.

Final response only when blocked or at release candidate: report milestone, checks, commits, PR, status-file state, blocker/decision URL if any, and exact next action for Mathieu.
```
