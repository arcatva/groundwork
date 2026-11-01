# Review before you push

Three reviewers, each with its own question, so one kind of problem does not crowd out another.
Paths in the table are under `standard/`.

**Safety.** The review changes no file outside `.groundwork/`, and posts nothing: you draft, the
user decides what to post. Checking out someone else's PR runs its code: ask first.

<!-- gen:reviewers -->
| reviewer | its question | its files |
|---|---|---|
| Claims | Does the change do what its description, ticket, docstrings and comments say, everywhere they say it, and is every product choice it makes settled? What else must change with it? Does the PR description follow the repo's PR template, where it has one? | `functional/larger-defects.md` |
| Behaviour | Does the code behave right in every case its inputs can take? | `functional/check.md`, `functional/logic.md`, `functional/interface.md`, `functional/timing.md`, `functional/resource.md`, `functional/support.md`, `functional/tests.md` |
| Design | Does it follow the repo's rules and the repo's own way, and is it easy to read and change? | the repo's agent instruction files and rules docs for each touched path, then `evolvability/textual.md`, `evolvability/supported-by-language.md`, `evolvability/organization.md`, `evolvability/solution-approach.md` |
<!-- /gen -->

## 1 · Gather

- The diff, from the merge base to the head; for your own change, uncommitted work included.
- The PR description and discussion, through the code host's CLI or API; for your own change, what
  you would write in it.
- The ticket the branch, commits or description name.
- The PR template of the PR's own repo, at its head.
- What the repo's type checks and linters print for the touched files.

Review someone else's PR, and run anything that needs another commit or a clean tree, in a separate
worktree, never by stashing or switching the user's tree. Without one, a claim about code outside
the diff is unverified.

## 2 · Run the three, then wait silently for all three reports

Start the three as subagents at once, however small the diff, on a mid-size model, or the most
capable one for a PR the user marks high-stakes: a reviewer that shares your context inherits what
you already believe. Only an agent that cannot start subagents runs them one after another, each
from a fresh read of its own prompt.

Each prompt holds the diff, the description, the ticket and discussion, the docs found, its
question, and the paths of what it reads in full first: the sections "Evidence", "Authority" and
"Enumeration" of `SKILL.md`, the Finding and Question shapes of `contracts.md`, and its rule files.
Then this brief:

> Read the rules above, then the diff, and as much of the repo as you need: the callers and readers
> of what changed, and every place a claim covers, not only the diff. Go through each rule above
> against the change. Report problems in your question first, then any other you notice. Report only
> what needs to change: leave out what you checked and found fine. Before you report that something
> is unused, missing or wrong, find it in the code, including the tests; only a product choice the
> code cannot settle goes to the author, in the Question shape; a fact stays a finding marked
> unverified. Write each in the Finding shape; the most direct fix is often to remove, rename, make
> it the default, or move. Skip what the discussion resolved and what the linters enforce; Gather
> ran them. Write your report to `.groundwork/<reviewer>.md`; change no other file.

The Design reviewer's prompt adds:

> Before you report, list every name the diff adds: each function, type, field, parameter, variable,
> and any UI name (prop, component, CSS class). For each, one line: what it holds or does, how the
> repo names the same role elsewhere with a `git grep` count, and whether the word already means
> something else in the repo or in the same file. Put the list at the end of your report under
> `## Name inventory`; a name that fails also goes in your findings.

## 3 · Report

Show the user the findings under `## Claims`, `## Behaviour`, `## Design`, in a report as
`contracts.md` shapes it. An item two reviewers raise is listed once, where it fits best; one the
code disproves goes to the log; nothing else is merged, reranked or dropped.

**Gate.** Each finding the user sees has its four parts, or went back to its reviewer.
