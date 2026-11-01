---
name: groundwork
description: Takes a code change from request to reviewed pull request (PR or merge request) - frame the requirement, plan the PRs, build against a defect-type standard, review with three parallel reviewers, ship - with every claim tied to evidence and every product decision tied to its source. Use when starting a feature, fix or refactor, when splitting work into PRs, before pushing or opening a PR, or when asked to review a pull or merge request. Not for explaining code, or for edits that will not become a PR.
---

# Groundwork 0.4.0


You take a change from a request to a reviewed pull request (PR; a merge request is the same thing).
Three principles apply at every step: evidence, authority, and enumeration. The rule files in
`standard/` apply the principles to each kind of code, by the defect types of Mäntylä and Lassenius
(2009), as refined by Beller et al. (2014): `evolvability/` for what makes code hard to read and
change, `functional/` for what makes it behave wrongly. Most hold a gate: something you produce in
your notes or your reply, never in the code, before you move on.

## Evidence

**Rule.** Never state a guess as a fact. Every claim, and every assumption the code makes about how
things are, names its evidence.

| Claim about | Evidence |
|---|---|
| Our code | The function body and its call chain, read or run. Never the name or the signature alone. A green test does not prove a branch is reached: check what the real callers pass. |
| Reach of a change or a risk | Every reference you grepped, and every consumer outside the repo (services, clients, stored data, later PRs in the stack), each found or marked unverified. |
| Real data | A count from a copy of production data you are allowed to query, with the date, run once the user agrees, on a table you checked is not truncated. |
| A third-party component | Its own docs at the version the repo uses. A tool that reports "not found" in a large document proves nothing. Never call a third-party service yourself. |
| How the business works | The code that carries it out, then the repo's docs, then the team's written records, searched for the term. Where the code and a document disagree, both are named. |
| How others do it | Their own pages, not a search result's summary of them. |

Where evidence is out of reach, say "unverified", where you looked, and what would verify it. A
claim the change makes (a docstring, a comment, the PR description) is checked against the code the
same way, including what the description leaves out.

**Gate.** A claim with no evidence is not written. Code does not rely on an assumption until its
source is found or the user accepts it as unverified. Code that contradicts a fact found is a
defect, whatever a source of what the code should be says.

## Authority

**Which source wins**, about what the code should be, highest first:

1. The repo's written rules: its agent instruction files (`AGENTS.md`, `CLAUDE.md` and the like),
   its rules docs, its linter config, a deprecation mark in the code.
2. A decision the ticket, the design, the PR's discussion or a recorded team decision states. The PR
   description is the author's claim, checked like any other, not a decision that settles a
   point.
3. What the repo does, counted with `git grep` at each directory from the changed file's up to the
   whole repo, the nearest sibling files among them. The file you copied from is not a source on its
   own: it may be the old way.
4. These rule files. Never present one as the repo's rule.
5. Your own reading, reported as that.

Report a difference with its source and rank, unless a higher source sides with the code. Code that
breaks a written rule is fixed, or says why the rule does not apply. Two sources of one rank that
differ (two directories' counts) are both named and the user decides; neither silences the other. A
rule outranks a count: the most common form is often the old one.

**Product decisions** (what the feature does for the user, new or changed: who can act, a limit, a
default, what the user sees) are not yours to make. Each one comes from a quoted ticket line, a
linked design or decision, or the user. The scope of a PR, and how work splits into PRs, is such a
decision. A choice the user has made is a constraint, not an open risk.

**Gate.** Before encoding a product rule, name its source; with none, ask. Put a choice as: what the
code does now (`path:line`), what the sources say, two options or more shown on a concrete value,
and the one you recommend. A concern the user has ruled on comes
back only with new evidence.

## Enumeration

**Rule.** Before you judge what the change touches, list its whole space, and give each member an
outcome: every value an input can take, every reference to what changed, every state a thing can be
in, every writer of shared state, every place a setting lands. A member with no outcome is a defect
or a question; one the change leaves as it was is out of scope, named once. Each rule file's gate
names the space to list for its kind of code.

## The flow

<!-- gen:steps -->
| step | produces | file |
|---|---|---|
| Frame | A brief: the problem, the behaviour as cases, what is out of scope, each with its source | `flow/frame.md` |
| Plan | The PRs, in order, each with its cases, its reach and its first failing test | `flow/plan.md` |
| Build | Code and tests, each rule file's gate met, the repo's checks run | `flow/build.md` |
| Review | Findings from three reviewers, each with its evidence | `review.md` |
| Ship | A PR description that matches the diff, and the user's go-ahead | `flow/ship.md` |
| Learn | Each miss and false alarm from people's review, as the smallest change to the skill | `flow/learn.md` |
<!-- /gen -->

Read `contracts.md` and the run log before the first step. Each step ends at its gate, with a
report and a log entry in their shapes. A **small change**, one behaviour with no new public name,
schema, setting or product rule, starts at Build.

Ask the user before you push, open a PR or call the work done whether to run the review; to review
anyone's PR, follow `review.md` at its head. After people review a PR, offer Learn.

## Which rule file to read

Read each file the first time it applies in a session, before writing the code it covers.

<!-- gen:rules -->
| file | read when |
|---|---|
| `standard/evolvability/textual.md` | a name, a docstring or comment, a message, a PR description |
| `standard/evolvability/supported-by-language.md` | a type, a public symbol, a signature |
| `standard/evolvability/organization.md` | a function, constant or type, and where it goes |
| `standard/evolvability/visual-representation.md` | never: run the repo's formatter and linters |
| `standard/evolvability/solution-approach.md` | a helper or mechanism, a call to existing code, a literal value |
| `standard/functional/resource.md` | an object, a query, a write |
| `standard/functional/check.md` | a branch, filter, lookup or default |
| `standard/functional/interface.md` | a call or a signature |
| `standard/functional/logic.md` | a comparison, a calculation, an expensive call |
| `standard/functional/larger-defects.md` | a change that must fit the whole: the feature finished, each state of a screen, code outside the diff still right |
| `standard/functional/timing.md` | a write, a job, a webhook |
| `standard/functional/tests.md` | a test, or behaviour that needs one |
| `standard/functional/support.md` | a migration, a setting, a dependency |
<!-- /gen -->
