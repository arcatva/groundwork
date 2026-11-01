# Handoff

Where the work stands, for whoever picks it up. Read `AGENTS.md` and `CONTRIBUTING.md` first; the
history of every round is in `evolution.md`.

## How the project evolves

Each round: a peer review by a separate agent (it reads the skill, the newest eval traces and the
harness, and is never shown the author's conclusions), the changes taken from it, then the evals
with and without the skill to catch a regression. The core principles do not change: ranked
sources, evidence for every claim, product decisions belong to people, a standard by defect type
with a gate per file, a review by three reviewers. The skill stays within its word budget;
principles over case lists.

## Open, in order

1. **The Go harness's first real eval run half failed.** After round 8, `groundwork eval` gave
   usable scores for two scenarios a side (with the skill: ambiguous-feature 9/9, review-planted
   11/11; without: ambiguous-feature 1/9, review-scope 4/4, small-bug 5/9). The rest were either
   "did not finish" although the run's result said success, or ungraded because the grader's reply
   held no JSON. The Python harness never showed either over hundreds of runs, so the cause is in
   the Go runner (`internal/agent`, `internal/judge`). It is intermittent: a rerun of small-bug on
   2026-11-01 finished and scored 9/9. Both errors now record the exit code, the end of stderr and
   the end of the grader's reply, so the next one explains itself in its record.
2. **Round 8 needs its eval** once item 1 is fixed: `groundwork eval`, then `--baseline`.
3. **The comparison with other review skills has no result yet.** 18 of 50 reviews (5 suites x 5
   held-out Django PRs x 2 runs) ran on the Python harness before the run was stopped; they are not
   published. Rerun it whole with the Go harness, so one harness and one skill version make every
   number: `groundwork compare run --pr 22169,22176,22137,22179,22095 --runs 2`, then `compare grade`
   and `compare report`, and fill in `comparison.md`. It runs one review at a time (about 4 to 5
   hours, about 80 US dollars).
4. **Machine-checked contracts, the half that changes the skill.** `groundwork check report|log`,
   the schemas under `skills/groundwork/schemas/` and the generated tables exist and are tested. The
   skill does not yet tell reviewers to write their reports as JSON in the reviewer report schema,
   nor the run log to be `.groundwork/log.jsonl`. That is a skill change: peer review and evals
   first, and pay for the words.
5. **Release 0.5.0** once 2 to 4 hold: bump every manifest and `SKILL.md`'s heading together, move
   the CHANGELOG entry, tag `v0.5.0`.

## Facts that cost time to find

- Claude Code's sandbox gives every run the same temp folder (`/tmp/claude-<uid>`), whatever the
  environment or settings say. Hence one sandboxed run at a time, the lock, and the archive of what
  each run leaves there.
- Some machines load plugins in every session that the command line cannot turn off; one blocks
  `python -c`. Record it with any published number.
- A sandbox notice repeats the read-deny list, so a leak check that scans raw output flags every
  run; the leak check reads tool results instead.
- Never let a reviewer read the human review: it lands in session transcripts, which other runs can
  read. Peer reviewers are told not to open `evals/compare-data/` or `evals/results/compare/`.
