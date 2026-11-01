# Plan

Turn the brief into PRs, in order, each one a reviewer can read in one sitting.

1. Find the reach, as the gates of `standard/functional/larger-defects.md` and
   `standard/functional/support.md` list it.
2. Propose a split, then ask: how work splits is the user's decision. A good split gives each PR one
   purpose and lets it ship safe alone (unused, behind a flag, or compatible with the version before
   it), with what others depend on first.
3. Write the plan in the Plan shape of `contracts.md`.
4. A risk with no evidence is a question or a check to run.

**Gate.** Each case maps to a PR, each reference in the reach is changed or says why not, and the
user agrees to the split.
