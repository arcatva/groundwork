# Logic

Whether the code computes the right thing.

## Compare

**Rule.** A comparison uses the right operator, boundary and type: `<` against `<=` at a limit,
dates against datetimes in one timezone, enums against enums.

## Compute

**Rule.** A value that must be exact (money, counts, IDs) uses an exact type (integer minor units or
a decimal), never a binary float, rounded where the domain rule says; units and precision travel
with it.

## Wrong Location

**Rule.** A statement sits where it takes effect for every path that needs it: a check before the
write it guards, a step inside the loop or transaction it belongs to.

## Algorithm/Performance

**Rule.** The same expensive call (a query, a network call) is not made twice for one answer; such a
call inside a loop is batched into one.

**Gate.** For each comparison, computation and expensive call the diff adds, show it on a concrete
value at its boundary, and count the calls it makes per operation or item.
