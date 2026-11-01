# Check

Whether the code checks what it reads: return values, variables and input.

**Rule.** For each value the code reads, know every case it can take: absent, zero, empty, each
member of a closed set, each value of every dimension the domain varies by (a currency, a region), a
duplicate, and a value written before the change or read by code from before it. Each case has an
explicit outcome, and one the code does not handle is either handled or stated as out of scope.

## Check Function

**Rule.** A call that can fail or find nothing has its outcome checked. A case the types and callers
allow but the logic does not expect fails loudly and names what it was looking at; it is never a
silent `return`, a default nobody chose, or an unchecked first-item lookup.

## Check Variable

**Rule.** A check for absence does not also catch a real value. Where a value's type admits `0`,
`""`, `[]` or `false` as a real value, test absence as absence, the way the language does, and
default on absence only. Where it admits none of them (an object, a date, an id), truthiness says
the same thing: write it the way the repo does. No non-null assertion (`x!`) on a value that can be
absent.

## Check User Input

**Rule.** Input from outside the program is validated where it enters, in the repo's schema or
parser where there is one.

**Gate.** For each branch, filter, lookup and default the diff adds, list the cases its input can
take and where each one goes. A case with no outcome is a finding. Each truthiness check on a value
that can be `0`, `""`, `[]` or `false` names what that value means at that line.