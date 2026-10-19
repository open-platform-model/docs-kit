## Context

`openspec validate --all --strict` under 1.14.1 reports 47 requirements over 500 characters in
17 main specs of this repo (counted 2026-10-19 on `origin/main` 28a5f5d, with
`2>&1 | grep -c WARNING`; the per-spec counts equal those of the swarm's inventory). The length
is the requirement body: the lines between `### Requirement:` and the first line that starts with
`#`, without blank lines, fenced lines and `**Key**:` lines, joined by `\n`. Scenarios do not
count. The check runs on main specs and on ADDED blocks, and not on MODIFIED blocks. A MODIFIED
body that stays over 500 therefore passes `validate <change> --strict`, archives, and then fails
`validate --specs --strict`. This repo has no active change, so no rewrite collides with one.

The change is text only. It has no trust boundary, credential, personal data or code path, so it
has no security section beyond this line: the only risk is a lost rule, and the ledger and the
review checklist guard that.

## Research & Decisions

### Which form per requirement
**Context**: Each of the 47 requirements needs a rewrite that keeps every rule and every
scenario, and the validator does not compare scenario text.
**Explored**: The four forms of the approved guide (a: MODIFIED with a shorter body and more
scenarios; b: MODIFIED plus ADDED; c2: REMOVED plus ADDED under new names; d: RENAMED plus
MODIFIED) and their unsafe pairs, run in scratch copies by the swarm.
**Options considered**:
1. Form a everywhere - keeps names and scenarios safe, but 40 of the 47 requirements are labelled
   "several behaviors" and a body of several SHALL clauses stays untestable one by one.
2. Form c2 everywhere - gives one behavior per requirement, but it is the only form that may lose
   a scenario with no error, and it breaks any reference to the old header.
3. Choose per requirement from what the excess is (the guide's table).
**Decision**: Option 3.
**Rationale**: The table of the guide fits this repo:

| The excess is | Form | Typical docs-kit case |
| --- | --- | --- |
| Examples, lists, formats, rationale around one behavior | a | "format" only: `bundle-format`, "One version's docs bundles never overlap", "Authored docs pages ship in docs bundles" |
| Several behaviors, and the old scenarios fit the parent that stays | b | A parent rule plus one or two split-off behaviors |
| Several behaviors, and the old scenarios belong to different behaviors | c2 | Long "several behaviors" requirements such as the `opm-docs-cli` commands and `bundle-pull` pull config |
| A name that must change, with no rule change | d, alone | Not expected; no rename is planned |

The kind labels of the inventory are a heuristic: the implementer reads the text before it
chooses. A default or an exact value may move from a body to a scenario THEN line (keeping its
SHALL). A clause that adds or changes a condition stays in a body. A separate behavior goes to
form b or c2.

### The batch order
**Context**: Batches are whole specs; B1 has 28 requirements, B2 has 19.
**Options considered**: (1) B1 then B2 (first-fit decreasing, as the inventory built them);
(2) B2 first because it is the smaller. 
**Decision**: B1 then B2, one tasks.md section each, then the `--strict` section.
**Rationale**: The `--strict` flag can return only after both batches pass, so the extra is last
and carries the final gate. Each section ends green and closes with one commit
(Principle VII). The order inside B1 starts with `bundle-pull`, the largest spec (9), whose
two pull requirements overlap in topic, so their rules are read together first.

### The extra: `--strict` back in `openspec:check`
**Decision**: One section after both batches (success criterion 4 of the swarm). It edits the
command and its `desc` in `Taskfile.yml`, the commands-table row in `AGENTS.md`, and the
Validation Gates item 4 in `openspec/config.yaml`. CI runs `task check` (`.github/workflows/ci.yml`)
and the CLI pin is already 1.14.1 there, so CI needs no edit.
**Reversibility**: high (a one-line revert).

## The rewrite rules of this repo

From `openspec/config.yaml` rules for specs and the guide:

- Every body keeps RFC 2119 keywords (SHALL or MUST). A clause moved to a scenario THEN line
  keeps its SHALL there.
- A citation (`DESIGN decision N`, `docs-kit C5`, an enhancement id beside a reason in words)
  stays in the body of the block that carries the rule it supports, never in a header, and never
  alone as an enhancement id.
- Requirements describe observable behavior: files, bytes, tags, exit codes, messages. A new
  scenario uses only words and values of the old body; the plain negation of an old condition is
  allowed. A scenario that adds a case not in the old text is a question for the human.
- The loopholes are forbidden in a changed block: no `**Key**:` line (only `**Reason**:` and
  `**Migration**:` in REMOVED), no heading of level 5 or deeper, no fenced block in a body.
- A new requirement name is unique in its spec. Before a split or a c2 the implementer greps the
  repo, outside `openspec/changes/archive/`, for the old header text (including `DESIGN.md`,
  `docs/`, `README.md`, tests and testdata) and fixes or reports every stale reference.
- Every ADDED body is 500 characters or less, has SHALL or MUST, and has at least one scenario
  with a body. A MODIFIED body is also kept to 500 or less, although validate does not check it.
- A touched spec must have a real Purpose. None of the 17 specs has the placeholder today (the
  47 warnings are all length warnings); the implementer re-checks it in the gate.
- Commit types: `docs` for spec text, `ci` or `chore` for the gate. Scope `openspec`. None
  releases `opm-docs`.

## The scenario ledger

For each rewritten requirement the implementing unit adds a table here, in the section of its
batch, before its commit:

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |

- Forms a, b, d: every old name stays under the same requirement, byte for byte.
- Form c2: every old name reappears in exactly one new requirement, with its WHEN and THEN
  unchanged. OpenSpec does not check this, so the reviewer checks it by name.
- A second table lists each clause of the old body that moves (old clause, new place), including
  citations, with the SHALL strength unchanged.

Ledger B1: filled in during section 1.

Ledger B2: filled in during section 2.

## The gate

Run in a scratch copy of `openspec/`, never in the worktree. Set before the run:

- `WT` = `/var/home/emil/dev/open-platform-model/agent-swarm/openspec-requirement-split/worktrees/docs-kit-split-long-requirements`
- `CH` = `split-long-requirements`
- `S` = a new folder under the session scratchpad (for example `.../scratchpad/T4.3-gate`)
- `N` = 28 after section 1, 47 after section 2; `LAST=1` only in section 3
- CLI: `openspec` 1.14.1 (the pin of `Taskfile.yml`), `jq` installed.

The block is the one of guide section 6 unchanged (`errs()`, base copy from `git archive main`,
`validate $CH --strict`, `archive $CH --yes`, `validate --all --strict`, the "Archive would
refuse" check, the new-ERROR set check, then the `N` check or the `--specs --strict` check). It
passes only on `GATE PASS` with exit 0. Because this repo has no other open change, in section 3
`validate --all --strict` also exits 0. Count by hand with
`openspec validate --all --strict --no-interactive 2>&1 | grep -c WARNING`; it prints 47 on
`main` and must print 0 after section 2 in the scratch copy.

The review checklist of the guide (scope, meaning kept, scenario text byte equal, no invented
rule, ledger, no loophole, lengths, forms, one behavior each, repo rules, references, Purpose,
gate rerun, commits) applies to each section.

## Durable decisions

- `--strict` is on in `task openspec:check`: lands in `Taskfile.yml`, `AGENTS.md` and
  `openspec/config.yaml` (section 3).
- The ledger and the choice of form per requirement: stay with the change.
