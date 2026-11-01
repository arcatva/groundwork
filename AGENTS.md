# Working on groundwork

This repo is the `groundwork` skill, its manifests for each agent, and `groundwork`, the Go command
that checks the skill and evaluates it. These notes are for changing the skill or the command, not
for using the skill in another project.

- The skill is `skills/groundwork/`. `SKILL.md` holds the three principles (evidence, authority,
  enumeration), the flow and the indexes; `flow/` and `review.md` say what to produce at each step;
  `contracts.md` gives every hand-off its shape; `standard/` holds every rule.
- A rule lives in exactly one file under `standard/`, the file of its defect type. The flow and the
  review only point at rules.
- The skill names no company, product or vendor a reader would have to use, and no agent's tool
  names. Keep words private to you in an untracked `.neutral-words` file; `groundwork lint` checks
  every tracked file against it.
- After every change: `go test ./...` and `go run ./cmd/groundwork lint`. CI runs both.
- A change to a rule comes with an eval scenario in `evals/scenarios/` that shows it, and the
  scores from `go run ./cmd/groundwork eval` (and `--baseline`) before and after. The Learn step in
  `flow/learn.md` is how a missed review comment becomes such a change.
- Bump `version` in every `.*-plugin/plugin.json` and in `SKILL.md`'s heading together; lint fails
  when they differ. Then tag `vX.Y.Z` to release the command's binaries.
- Evals and comparisons run sandboxed agents that share one temp folder: one harness run at a time
  (a lock enforces it).
