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

Official Arch repository upgrades must remain supported, complete transactions. AUR packages may be deferred, together with dependants that cannot safely proceed; the reason must be shown.

## 4. Recipe identity and acquisition

For each AUR `pkgbase`, AURoscope collects recipe data without sourcing the `PKGBUILD`:

- candidate Git commit OID;
- `PKGBUILD`, `.SRCINFO`, `.install` files, patches, units, rules, and other tracked files;
- relevant maintainer and source metadata;
- hashes of all inspected files.

The candidate commit is an immutable identity, not a risk signal. It is used to define the diff, cache inspections, bind human approval, and prevent time-of-check/time-of-use substitution. File hashes additionally detect local or workspace modifications.

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

## 8. Generic terminal review

Reports use ordinary text, Markdown, unified diffs, and normal files. The review command resolves the viewer in this order:

1. `$VISUAL`;
2. `$EDITOR`;
3. a terminal pager fallback.

After the viewer closes, a plain terminal menu offers approval, further inspection, deferral, rejection, or cancellation. No Vim, Neovim, Emacs, or other editor plugin is required.

If the user modifies recipe content, the prior identity and approval are invalidated. Modified content must be rehashed and reinspected before it can be built.

## 9. Execution and TOCTOU guard

After human decisions, AURoscope delegates the executable plan to Paru/Pacman. Deferred or rejected AUR recipes and impossible dependants are omitted with an explanation; official repository upgrades are not fragmented into unsupported partial upgrades.

Paru's `PreBuildCommand` remains as a narrow guard only. Immediately before each build, it verifies that the recipe commit and file hashes exactly match a live AURoscope approval. Any divergence aborts the build and returns the recipe to review.

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

## 11. Initial scope

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
- replacement of Paru's resolver.

## 12. Acceptance criteria

AURoscope is not useful until tests demonstrate that:

1. bare `auroscope` can replace bare `paru` in the normal upgrade workflow;
2. Paru's numbered interactive selection is preserved;
3. official packages proceed without AUR-specific friction;
4. AUR recipes and required dependencies are inspected before build;
5. deterministic and LLM evidence remain visibly separate;
6. all consequential interactive decisions belong to the user;
7. held recipes and dependency effects are readable through `status`, `held`, and `explain`;
8. approval applies only to the exact inspected commit and hashes;
9. a recipe changed after approval is stopped by the pre-build guard;
10. cancellation leaves no reusable floating approval;
11. end-to-end fixtures run in a disposable Arch environment without altering the real workstation.

## 13. Relationship to the previous project

AURoscope is a from-scratch successor to [2027a/paru-llm-audit](https://git.2027a.net/2027a/paru-llm-audit). The old project is a reference for lessons, test fixtures, and scanner ideas only. No production code or hook-centered architecture is inherited implicitly.
