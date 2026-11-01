# Organization

Where the code sits and how it is laid out. Each module holds one subject.

## Move Functionality

**Rule.** Put each piece of code beside the thing it is about. Behaviour that reads, computes or
formats a type is a method of that type. A layer that only connects (a route, a view, a handler, a
component) calls the code that does the work. Where the repo's rules say where a change of state
goes, code goes where they say. A module holds one subject: code the diff adds to a file for a
reason unrelated to the rest of that file goes to a module of its own, and edits the change makes
the same way across many files mean the thing they change belongs in one place. A caller does not
walk another object's internals (`a.b().c().d()`); the first object offers what the caller needs. A
function or class that only passes calls on is removed and its callers call the target. A subclass
that overrides or ignores most of what it inherits holds the other class instead.

## Long Sub-Routine

**Rule.** A function that does several things in turn, or is too long to take in at once, is
split: each part becomes a function named for what it does.

## Dead Code

**Rule.** Code no path reaches, commented-out code, and symbols, fields and parameters nothing reads
are removed, unless a later PR the plan names reads them and the description names it.

## Duplication

**Rule.** The same lines written twice become one, and so does the same `switch` or `if` cascade on
the same value in two places.

## Complex Code

**Rule.** Nesting, flags and conditions are no deeper than the logic needs: return early, name a
compound condition, split a branch that does two things. Write each thing the most direct way: a
helper, a variable or a mechanism stays only when it reads more simply than the expression it
replaces, and a function with one caller and a body of a few lines is inlined, unless it is
behaviour of a type (Move Functionality). A type, parameter or layer added for a case the change
does not have is removed.

## Statement Issue

**Rule.** Each statement does one thing; a statement with a side effect is not hidden inside an
expression.

## Consistency

**Rule.** New code takes the form the repo already uses for the same construct. A new set of files
that parallels an existing one (a second flow, section or variant) takes its layout and its names,
or extends it as a variant.

## Grouping

**Rule.** Related declarations sit together in the order the file and its siblings already use;
constants at the top.

**Gate.** For each new function, write what it reads, what it computes and which layer it sits in,
and whether that layer is where the repo keeps such code. For each new constant, type and
docstring, name the thing it is about and check it sits in that thing's class or module, in the
place its siblings take there. For each new helper, variable and mechanism, write the direct
form beside it; when the direct form reads as well, use it.
