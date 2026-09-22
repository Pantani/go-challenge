# File Store Capability Challenge

This challenge is a small extraction of Glovebox's file storage abstraction. It includes a provider interface, an S3-shaped implementation, and an in-memory mock used by tests.

## Business Problem

The application needs to copy an existing stored file to a new key without exposing provider-specific behavior to callers. Add the capability to the file store abstraction and implement it for every provider.

## Instructions

* [x] Add the new copy operation to the appropriate provider interface.
* [x] Implement the operation in the S3-shaped provider and the in-memory mock.
* [x] Preserve the existing error contract for missing files and destination conflicts.
* [x] Add unit tests for success, missing source, and destination conflict behavior.
* [x] Ensure the module and demo compile and all tests pass with `go test ./...`.

## Implementation notes

`FileProvider.Copy` duplicates a stored object without exposing provider-
specific behavior. Both the in-memory mock and S3-shaped adapter preserve the
source bytes and content type, return errors compatible with `ErrNotFound` for
a missing source, and return errors compatible with `ErrFileExists` for an
occupied destination.

The S3-shaped adapter uses a narrow `ObjectAPI`: it probes the destination,
then calls `CopyObject`. That check is best-effort; a real concurrent writer
can still create the destination between the two operations.

Run the module checks with:

```sh
cd filestore
go test -race ./...
go vet ./...
```
