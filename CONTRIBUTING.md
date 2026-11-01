# Contributing

Thanks for helping. Two kinds of change come in: to the skill (the markdown under `skills/`), and to
the `groundwork` command (the Go under `cmd/` and `internal/`).

## The most useful report

A review finding the skill missed, or one it got wrong, on real code. Open an issue with the
"Missed or wrong review finding" form: what a person caught (or why the finding was wrong), the
smallest code that shows it, and your run log entry. That is the input the skill's Learn step
(`skills/groundwork/flow/learn.md`) turns into a change.

## Changing the skill

1. Find the principle the case breaks (evidence, authority, enumeration in `SKILL.md`) and the rule
   file of its defect type.
2. Change as little as you can, in this order: fix the step that failed to apply what is there;
   reword an existing sentence to cover the case; delete or narrow a sentence that raised a false
   alarm; and only then add one sentence that states a principle, never a single case.
3. Add a scenario in `evals/scenarios/` whose rubric names the principle, not the skill's words.
4. Run the evals before and after, with and without the skill, and put both scores in the PR:

   ```sh
   go run ./cmd/groundwork eval
   go run ./cmd/groundwork eval --baseline
   ```

   Evals call Claude Code headless and cost a few dollars a round. They run one at a time.

The skill has a word budget (`internal/lint`). A change that needs more words removes others, or
raises the budget with the reason in the commit message.

## Changing the command

```sh
go test ./...                        # unit tests, and the repo's own rules
go run ./cmd/groundwork lint         # the same rules, as CI runs them
gofmt -l . && go vet ./...
```

The command uses the Go standard library only. Records it writes (`evals/results/`) keep their JSON
field names, so older results stay readable.

## Releasing

Bump the version in every `.*-plugin/plugin.json` and in `SKILL.md`'s heading (lint checks they
agree), add the entry to `CHANGELOG.md`, and push a tag `vX.Y.Z`. The release workflow builds the
command for macOS and Linux and attaches it to the GitHub release.
