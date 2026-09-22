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

## Copy Operation

`FileProvider` exposes a `Copy` method alongside `Get`, `Set`, `Purge`, and `Move`:

```go
Copy(ctx context.Context, oldFilename, newFilename string) error
```

Both providers implement the method, but only `mock.Client` gives it real behavior:

* **In-memory mock** (`mock.Client`) — duplicates the file stored under `oldFilename` to `newFilename` into a new, independent object, leaving the original untouched, so closing or reading one copy never affects the other. Enforces the error contract below.
* **S3-shaped client** (`s3.Client`) — a no-op stub, consistent with its other methods: it ignores both filenames and always returns `nil`, without creating a destination or checking whether either file exists.

### Error contract (`mock.Client` only)

* `filestore.ErrNotFound` — `oldFilename` does not exist.
* `filestore.ErrFileExists` — `newFilename` already exists.

## Running the tests

```bash
go test ./... -race -cover
```

`filestore/mock` and `filestore/s3` are both at 100% statement coverage. `filestore/cmd/filestore` factors its logic into a `run()` helper so the demo's behavior is covered; the `main` wrapper itself is intentionally left untested, since it calls `log.Fatal`.
