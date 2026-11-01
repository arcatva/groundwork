# Supported by Language

What the language's own features say about the code: types, immutability, visibility, parameters,
references.

## Element Type

**Rule.** A value's type says what it can hold: an enum for a closed set, an optional type only
where absence is a case, a dedicated type rather than a bare string or map where the repo has one,
and the narrowest exception that fits. Values that travel together, returned as a tuple or passed as
the same parameters to several functions, are one named type.

## Immutable

**Rule.** A value that does not change once set is declared so, the way the repo does it; a value
that does change is not.

## Visibility

**Rule.** A symbol is public only if something outside its module uses it; a private symbol used
outside is made public or moved beside its users.

## Void Parameter

**Rule.** A function that takes nothing declares so, where the language asks it to.

## Element Reference

**Rule.** Refer to a thing by its full name, the way the language and the repo do: a reference
the code already holds rather than a re-lookup by ID, and the repo's own import forms.

**Gate.** For each new type, signature and symbol, name the cases its type admits (each value of a
returned tuple among them), whether it changes once set, and the callers of each public name outside
its module.