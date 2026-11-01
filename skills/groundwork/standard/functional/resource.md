# Resource

How the code creates, holds and changes data and resources.

## Variable Initialization

**Rule.** Create an object with its fields in one call, not created and then assigned. A value is
set before every path that reads it.

## Memory Management

**Rule.** A query result, file or response the code holds is bounded: what can grow is streamed or
paged, not loaded whole into memory to be iterated once.

## Data & Resource Manipulation

**Rule.** Writes go where the repo's rules set. Work the store can do in one operation is done
there, not by loading items to loop and write back. A transaction is committed where the repo's
rules put it. A connection, file or handle opened is closed on every path.

**Gate.** For each new construction, query and write, say where its data comes from, how much of it
there can be, and where it is committed or closed.
