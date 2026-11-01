# Changelog

The skill's version is the number in `SKILL.md`'s heading and in every plugin manifest. Each round of
change, with its peer review and eval scores, is in [docs/evolution.md](docs/evolution.md).

## Unreleased

- The harness is now one Go command, `groundwork` (`lint`, `eval`, `compare`), with typed records,
  table-driven tests that break each repo rule once to prove its check catches it, and a lock that
  keeps two sandboxed runs from sharing a temp folder at once. The Python tooling is gone.
- CI (format, vet, tests, lint) and release builds for macOS and Linux.
- `skills/groundwork/index.json` is the one source of the flow, the reviewers and the rule index:
  `groundwork gen` writes the tables and the JSON Schemas from it, and lint fails on a hand edit.
  `groundwork check report|log` holds a reviewer's report or a run log to its contract and the flow.
- A question owes a recommendation even with no source; another author's description is held to the
  repo's template, where it has one; reviewers are awaited silently and write their own files under
  `.groundwork/`.

## 0.4.0

- The PR description is a claim, not a decision; what the change leaves as it was is out of scope,
  named once; only a product choice is asked, and a fact the code cannot settle stays a finding
  marked unverified; the user sees findings, not process.
- The three reviewers run as subagents however small the diff; anything that needs another commit
  runs in a worktree, never by stashing the user's tree; the change does no less and no more than
  asked.

## 0.3.0

- Three principles (evidence, authority, enumeration) in place of scattered case rules, and a word
  budget.
- Contracts for every hand-off (`contracts.md`); a run log that ignores itself; a three-state status
  for every report; numbered questions with lettered options; Build and Question shapes.

## 0.2.0

- The Learn step: misses and false alarms become the smallest change to the skill, with an eval
  scenario.
- Evals with and without the skill; plugin manifests for several agents; rules made language- and
  stack-neutral.

## 0.1.0

- First open-source version: frame, plan, build, review, ship, with a standard by defect type.
