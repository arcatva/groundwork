# Contracts

Every hand-off takes one of these shapes. Whoever receives one checks it against its shape and the
code, and sends back what is missing rather than filling the gap with a guess.

## What the user sees

Each step ends with one report, and nothing before it but questions that block the step:

```
**<Step> · <done | needs you: what | blocked: why>** (≤8 words)
<the step's output, in its shape below>
**Next:** <what happens next, or what the user must answer>
```

- The result first. What you did or dropped goes only to the log, unless it is evidence for the
  result.
- A table or a list wherever the content has parts; a short diagram (Mermaid or text) wherever
  three or more things depend on each other, such as the order of PRs.
- Each claim with its evidence inline (`path:line`, a count, a link).
- Leave out what the user can ask for: one screen where possible.

## Brief

**Problem:** one sentence, with its source.

| # | given | when | then | source |
|---|---|---|---|---|

`source` is a quote, a link, `user`, or `OPEN`. Then **Out of scope:** a list, each with who
decided.

## Question

Numbered, with lettered options, so whoever decides can answer `1a 2b`:

**1 · <the decision>.** **Now:** what the code does (`path:line`). **Sources say:** quoted, linked,
or nothing found.

| | option | what happens, on a concrete value |
|---|---|---|

**Recommend:** a letter and why, even with no source: you recommend, the user decides.

## Plan

| PR | purpose | cases | reach | first failing test | leaves for later |
|---|---|---|---|---|---|

With three PRs or more, a diagram of which needs which.

## Build

| case | test | failed before | passes after |
|---|---|---|---|

Then each check as `command→result`, and each not run.

## Finding

| file:line | what is wrong | the most direct fix | evidence, or a source and its rank |
|---|---|---|---|

 A reviewer's report is
`## <reviewer>` and its findings, then `## Name inventory` for Design.

## PR description

For your own PR without a repo template: **Intent**, **Changes**, **Left out**, **Verified**,
**Decisions**, in order; a small change needs only Intent, Changes and Verified.

## Run log

`.groundwork/log.md` in the repo: append to it as you go. Each
entry is headed `time · branch · commit · groundwork version` and holds the step's report, its gate
output, each send-back, each user answer or rejection, quoted, and each
correction: a step skipped, a gate passed without its output, a contract broken. The first entry
also writes `.groundwork/.gitignore` holding `*`, so git never sees the folder.
