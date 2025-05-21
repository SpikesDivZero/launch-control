# End-to-end tests

Tests in this folder are expected to test the only public interface, with limited exception.

The allowed exceptions are currently:

* `internal/lcerrors` is OK to use, as it provides our core errors that we expect to see.
