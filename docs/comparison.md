# Comparison with other review skills

How well does a review skill catch what a careful human reviewer catches? This page compares
groundwork's review with the review skills of three other suites, and with no skill at all, on
public pull requests. The harness is `groundwork compare` (see CONTRIBUTING.md); anyone can rerun it.

## Method

**Pull requests.** Merged Django PRs whose maintainers left inline review comments. Each is replayed
at the commit the first human review saw, so every reviewer sees what the humans saw. The repo is
rebuilt as two branches, `base` and `pr`, with their real commit messages and no other history, so
the commits that answered the review cannot leak in. The PR description is restored to its text at
the time of that first review. A symlink whose target has a `..` step is left out of the tree (in
Django, four documentation icons), and the shared Python environment is read-only to every run.

**Held-out.** groundwork was tuned on one PR (#22154). The table reports it apart, as "dev"; the
held-out PRs were chosen before their human review was read, and no rule was changed after it.

**Suites.** Each suite is loaded from all of its skills' markdown, as one plugin, and invoked through
its review skill. Its hooks and scripts are left out, so a suite that relies on them runs without
them here:

| suite | review skill | source |
|---|---|---|
| groundwork | `review.md` (three reviewers) | this repo |
| superpowers | `requesting-code-review` | obra/superpowers |
| matt | `code-review` | mattpocock/skills |
| pstack | `interrogate` | michael-denyer/pstack-claude, the Claude Code port of Cursor's pstack |
| baseline | none | |

`interrogate` reports only findings it proved by running code, and leaves out style, naming and
structure by design; points of that kind count against it here.

**Runs.** Headless Claude Code, the same model for all, in Claude Code's Bash sandbox: no network,
writes only in the work tree and the sandbox's temp folder. The sandbox gives every run the same
temp folder, so runs go one at a time; each run cannot read what was in that folder when it started,
and what it leaves there is moved out afterwards. Nothing the harness keeps (the human comments,
other runs' reports and leftovers) and no session transcript is readable from a run. A run is marked
contaminated and left out if any tool result it read holds eight words in a row of the human
review; so is one that trips over another run's files, or does not finish.

**Grading.** Blind and repeated. For each PR, a grader groups the human review's thread-opening
comments into points, one per distinct ask, and drops the trivial ones (typos, formatting, thanks).
Then, three times with a fresh shuffle, one grader call sees every suite's report for that PR under
letters, with the reviewers' own comments and the code, and says which points each report raised.
A "raised" counts only with a quote from that report, checked word for word, and a point counts as
raised when two of the three calls say so. Each other finding is judged valid (a problem the PR
causes or must fix), invalid or unsure.

**Score.** The share of human points a suite raised, per PR first and then averaged over PRs, so a
PR with many comments does not outweigh one with few.

## Limits

- A handful of PRs from one project. Treat a gap of a few points as noise.
- Human reviewers miss things too: a finding no human raised is not wrong, which is why other
  findings are judged separately.
- One model, one harness. A suite tuned for another agent may do better there.
- The runs could not be made fully clean: the test machine loads some Claude Code plugins in every
  session, and they cannot be turned off from the command line. One of them blocks `python -c`; it
  applies to every suite, but weighs most on one that proves findings by running code.

## Results

Not yet. 18 of 50 reviews ran before the run was stopped; they will not be published. See
[handoff.md](handoff.md) for the rerun.
