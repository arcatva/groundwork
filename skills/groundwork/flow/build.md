# Build

One PR at a time, in the plan's order.

1. Read the rule file for each kind of code you will write, from the table in `SKILL.md`.
2. Write the test that fails without the change, and run it: it fails for the reason the case gives,
   not for a typo or a missing import.
3. Write the most direct code that makes it pass and meets each gate.
4. Run the repo's tests, type checks, linters and formatter on what you changed. Report them in the
   Build shape of `contracts.md`.
5. Where the code shows the plan or the brief was wrong, stop and tell the user, with the evidence.

**Gate.** Each case has a test that failed before the change and passes after, each gate is met, and
the checks pass or each failure is reported.
