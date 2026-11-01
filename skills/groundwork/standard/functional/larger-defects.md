# Larger Defects

Defects that show only beside the rest of the system: the whole feature, the screen, and the code
outside the diff.

## Completeness

**Rule.** The change builds what its ticket and description ask for, no less and no more: a part
left for a later PR is named in the description, and a part beyond them moves to its own PR.

## GUI

**Rule.** A screen is consistent with its sibling screens, and offers the user the right options in
each state it can be in: loading, empty, error, not allowed.

## Check Outside the Code

**Rule.** Code outside the diff that the change makes wrong is changed with it: every reader of
what changed (callers, switches on its values, serialisers, UIs, other services) and every copy of
its shape. A change that narrows which values are valid counts
the stored values already holding a now-invalid one. A change that reaches past the PR's purpose (a
shared fixture, helper or mock others use) is named in the description.

**Gate.** Grep every reference to what the change alters and name, for each, whether this PR updates
it. For each screen the diff adds or changes, list its states and what the user can do in each.
