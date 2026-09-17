# File Store Capability Challenge

This challenge is a small extraction of Glovebox's file storage abstraction. It includes a provider interface, an S3-shaped implementation, and an in-memory mock used by tests.

## Business Problem

The application needs to copy an existing stored file to a new key without exposing provider-specific behavior to callers. Add the capability to the file store abstraction and implement it for every provider.

## Instructions

* [ ] Add the new copy operation to the appropriate provider interface.
* [ ] Implement the operation in the S3-shaped provider and the in-memory mock.
* [ ] Preserve the existing error contract for missing files and destination conflicts.
* [ ] Add unit tests for success, missing source, and destination conflict behavior.
* [ ] Ensure the module and demo compile and all tests pass with `go test ./...`.
