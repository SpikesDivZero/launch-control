# End-to-End Tests

The tests in this folder focus on testing the public interface from a user's point of view.

These may duplicate some ideas tested in finer detail elsewhere.
That's OK since the goal is to verify our public-facing expectations.

## Import Restrictions

With limited exceptions, we should not import anything other than the public package.

At this time, the only known exception to this policy is the utilities inside of `internal/testutil`.
