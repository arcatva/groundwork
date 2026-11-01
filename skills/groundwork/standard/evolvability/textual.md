# Textual

What a reader takes from the words: names, comments and docstrings, messages.

## Naming

**Rule.** A name says what the thing holds or does, and reads one way. A collection names its items;
a mapping names what it is keyed by; a word that can be read as both a verb and an adjective is not
used where the other reading fits. Names follow the domain's own words. A thing that is one-to-one
with a third party's concept takes that party's word for it, and the third party's name stays in the
code that talks to it. Words are spelled out (`max_retries`, not `mr`) unless the repo already
writes that abbreviation for that word. A test's name says the case it checks.

**Gate.** For each new name, write in a few words what it holds or does; if the name does not say
that, or a reader could take it another way, rename it. Use the word already in use for the thing,
with the sources ranked as "Authority" in `SKILL.md` ranks them, then the word other firms
use; a word already used for something else is not reused. The ticket's or description's name for a
thing gives the word, not its spelling: it does not make an abbreviation clear. A thing that
parallels one the repo already names takes that name, with what tells them apart.

## Comments

**Rule.** Prose a person reads (a comment, a docstring, a commit message, a PR description) earns
each sentence: cut what the code, the diff or the title already says, then say the rest in short,
plain sentences, one idea each, with the words the code uses.

- A comment says why, or what the code cannot show: a constraint, a source, a trap. It never
  restates what the line does.
- Prose describing a declaration is the language's doc comment (a docstring, JSDoc, a doc
  comment). Prose about a line inside a body is a line comment. Decide by what the prose describes,
  not by what follows it.
- A docstring describes the code as this PR leaves it, by what the code enforces: every value a
  field can hold, never a meeting or a later PR.
- No section banners, no slogan comments, no history ("changed from X", "now does Y"): that is the
  commit's job.

**Gate.** For each docstring and comment the diff adds, a reader with no context must be able to
say what it is for, and no sentence in it may be removed without losing something. Check PR
descriptions and commit messages the same way.

## Debug Info

**Rule.** An error, assert or log message is a sentence naming what was being looked at, with the
IDs needed to find it; what goes in it and where it goes are the repo's rules on error reporting and
logging. Debugging output left over from development is removed.

**Gate.** For each message the diff adds, say what a developer reading it in the logs would learn.
