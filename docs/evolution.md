# Evolution log

Each round: a peer review by a separate agent, the changes taken from it, the skill's word count,
and the eval score with and without the skill (the harness's `eval`, one run each). The core
principles do not change: ranked sources, evidence for every claim, product decisions belong to
people, a standard by defect type with a gate per file, a review by three reviewers.

Scores are comparable only within a round: rubrics grow as rounds add checks.

| round | change | words | with | without |
|---|---|---|---|---|
| 0 | First open-source version | 6,343 | | |
| 1 | Three-reviewer self-review: generality, followability, consistency; Learn step; evals | 7,141 | 27/27 | 17/27 |
| 2 | Bloat cut: case patches folded into three principles (evidence, authority, enumeration), example lists collapsed, Learn ordered delete-before-add, word budget test. Peer review then found ten losses of substance; all restored, each as a principle with one example | 6,201 | | |
| 3 | Contracts for every hand-off (`contracts.md`), run log, user-facing report shape; eval runner snapshots the skill, records its version and validates the grader's reply. Paid for by cutting Learn and the index | 6,300 | 31/31 | 19/31 |
| 4 | From traces: facts no longer forced into a source rank; three-state status (done, needs you, blocked); numbered questions with lettered options; Build and Question shapes; the run log ignores itself instead of editing `.git`; the log keeps raw reviews and user answers; review and ship gained gates; runs that did not finish are not graded | 6,300 | 33/36 | 20/36 |
| 5 | From traces: reviewers ran inline on small diffs, so the review says why they must be separate; raw reviews go to the log. Evals now run in Claude Code's Bash sandbox instead of a command allowlist; the three reviewers now run as subagents (3 Agent calls in each review trace) | 6,300 | 35/36 | 20/36 |
| 6 | From the round 5 review and a Django pilot: a finding keeps its location as the user sees it; a finding the code disproves goes to the log; anything needing another commit runs in a worktree, never by stashing the user's tree; the change does no less and no more than asked (the pilot's main human point). Evals: new files committed, traces de-noised and subagents marked, grader in an empty directory, outcome-only score | 6,299 | 38/40 (outcomes 31/33) | 22/40 (outcomes 22/33) |
| 7 | From the round 6 review: the PR description is again a claim, not a decision (the original's words); what the change leaves as it was is out of scope, named once; only a product choice is asked, a fact stays a finding marked unverified; the user sees findings, not process; the description shape is for your own PR; each reviewer keeps its raw report in the log folder. The comparison harness now runs one review at a time, since the sandbox gives every run the same temp folder | 6,298 | 39/40 (outcomes 32/33) | 23/40 (outcomes 23/33) |
| 8 | From the round 7 review: a question owes a recommendation even with no source (you recommend, the user decides); another author's description is held to the repo's template, where it has one; reviewers are awaited silently and write their own files under `.groundwork/`; four repeated sentences cut. Harness: the leak check reads what each tool returned, not the sandbox's own notices; the archive of leftovers is hidden from runs | 6,290 | (after the comparison) | |

After round 8 the evals moved to the Go harness, whose first run half failed (see
[handoff.md](handoff.md)); round 8 has no score yet.

## Tooling

- Rounds 1 to 8 ran on a Python harness. After round 8 the harness became one Go command,
  `groundwork` (`lint`, `eval`, `compare`), with typed records, a lock that keeps two sandboxed runs
  apart, and tests that break each repo rule once to prove its check catches it. Result files keep
  their field names, so earlier results stay readable.
