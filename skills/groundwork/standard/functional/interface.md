# Interface

How the code calls other code, and what passes between them.

## Function Call

**Rule.** A call passes what the callee expects, in the order and units it expects, and uses what it
returns; a call to a function whose contract the change alters is updated at every call site. A
function the change bends to a new purpose keeps a name, contract and other callers that still fit
it; when they do not, it is a new function.

## Parameter

**Rule.** A function takes values in the shape its callers already hold and converts at the boundary
that needs another shape; where the repo's rules set what a layer takes, it takes that.
What passes between modules is narrow: the values the callee reads, not a larger object that holds
them. A named type that holds exactly those values passes as one.
A function does not take a parameter it can derive from another it already takes.

**Gate.** For each new or changed signature, list its call sites and check each passes a value the
caller holds, in the shape and units the callee reads.
