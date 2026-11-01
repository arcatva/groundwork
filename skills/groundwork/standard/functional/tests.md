# Tests

Whether the tests prove the behaviour the change adds or alters.

**Rule.** Behaviour the change adds or alters has a test that fails without it, where the repo's
test rules put it, covering each case the change names. A test the diff changes still checks what
it checked before: an assertion removed or loosened, or a case dropped, is a finding unless the
behaviour it checked is removed too.

**Gate.** For each behaviour, name the test that fails without it. For each test the diff changes,
write what it asserted before and after.
