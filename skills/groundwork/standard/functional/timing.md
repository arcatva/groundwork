# Timing

Whether the code holds when things happen at once or out of order.

**Rule.** Read a value as it is when the code acts, not when it was captured: queued work runs later
than it was queued, a message can arrive twice or out of order, and two callers or threads can run
at once. A read-then-write on shared state another writer can change locks it first, or
writes it in one atomic operation that checks it. Work that runs again after a failure, a
retry or a duplicate leaves the same result.

**Gate.** For each write the diff adds, name what else can write the same state, and when; for each
queued job, incoming message and retry, say what happens when it runs twice or late.
