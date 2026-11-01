# Solution Approach

Whether the change solves its problem the way it should.

## Semantic Duplication

**Rule.** Code that does what existing code already does, written differently, is replaced by the
existing code: a method on the type, a shared helper, a library feature.

## Semantic Dead Code

**Rule.** A branch, guard or fallback for a case that cannot happen, because the types or every
caller rule it out, is removed.

## Change Function

**Rule.** A call to an old or deprecated function is changed to the one the repo now uses: the one a
rule, a `@deprecated` mark or the repo's own wrapper names.

## Use Standard Method

**Rule.** Do a thing the standard way the language, the library and the repo set: the enum member
rather than its string, the constant rather than a copy of its value or a bare number, the
language's error mechanism (an exception, a returned error) rather than a sentinel value, the
type's accessor rather than its raw fields.

## New Functionality

**Rule.** Where the change makes the code hard to extend, it adds the piece the next change needs:
the type, method or module the repo's other cases of the same kind already have.

**Gate.** Before writing a component, helper or type, search the shared code and the module it sits
in for one that does the same job. One that fits is reused, unless a rule marks it legacy. For each
call the diff adds, check the repo does not mark the callee old. For each string or number the diff
writes out, name the thing defined elsewhere it copies (an enum member, a constant, a route's path),
if any.
