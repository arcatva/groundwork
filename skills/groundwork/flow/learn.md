# Learn

Turn what went wrong into the smallest change to the skill, so the next run gets it right and the
skill does not grow. Run it after people review a PR this skill reviewed, or when the user corrects
how you followed it.

1. **Collect** each miss (a reviewer's comment, acted on, that the skill did not raise), false
   alarm (a finding the user rejected, with why) and correction (from the user or the run log).
   Leave out taste no source backs.
2. **Name the principle** in `SKILL.md` each one breaks, and the rule file of its defect type.
3. **Change as little as you can.** Take the first remedy that covers the case and its siblings:
   1. *Hand it over:* the gate covers it and the run did not apply it; fix the step that dropped
      it. No rule text changes.
   2. *Generalise:* reword a sentence already there to cover it, in no more words.
   3. *Delete:* a false alarm removes or narrows the sentence that raised it.
   4. *Add:* one sentence stating a principle, with at most one example, naming no repo, product,
      vendor or language; remove as many words elsewhere, or say why none can go.

   A lesson for one repo goes to that repo's instruction file, not the skill.
4. **Keep it caught.** Where the skill has evals, add a scenario and run it before and after. Its
   rubric names the principle, not the skill's words, so a reworded rule still passes. Report any
   other score that drops.

Show each change as a diff, with what it answers, quoted.

**Gate.** Each item has its principle, its remedy and a word count that did not grow, or why it had
to. Nothing is applied without the user's agreement.
