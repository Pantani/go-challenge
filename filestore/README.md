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

`Copy` duplicates `oldFilename` to `newFilename` (content and content type) as a new, independent object and leaves the source in place. `Move` is the same operation followed by removing the source. Both providers implement it with identical semantics:

* **In-memory mock** (`mock.Client`) — keeps objects in a map guarded by a `sync.RWMutex`, so it is safe for concurrent use and `Copy`/`Move` are atomic. Every `Get` returns a fresh reader over the stored bytes, so reading or closing one reader never affects the stored file or other readers.
* **S3-shaped client** (`s3.Client`) — talks to an `s3.ObjectAPI`, a narrow interface shaped after the S3 object operations it needs (`HeadObject`, `GetObject`, `PutObject`, `CopyObject`, `DeleteObject`, `PresignGetObject`). Plug in a thin adapter over a real SDK through `s3.Config.API`; adapters must translate the SDK's "key does not exist" error into `s3.ErrNoSuchKey`. When no API is configured the client reads as an empty bucket and rejects writes with `s3.ErrNoAPI`, so a missing adapter is caught at the first `Set` instead of silently losing data. `s3/s3test.MemoryAPI` is an in-memory `ObjectAPI` for tests. Because S3's `CopyObject` overwrites silently, the client probes the destination with `HeadObject` before copying; the two calls are not atomic.

### Error contract

The sentinels live in package `filestore` and may be wrapped with the operation and filename, so compare them with `errors.Is`.

* `filestore.ErrNotFound` — `Get`, `Move`, `Copy` and `GetPresignedURL` when the (source) file does not exist. `Purge` is idempotent and never reports a missing file.
* `filestore.ErrFileExists` — `Move` and `Copy` when the destination already exists, including moving or copying a file onto itself. Providers never overwrite implicitly; `Purge` the destination first.

When both conditions hold, `ErrFileExists` takes precedence: the destination is checked first so that S3 needs a single existence probe before `CopyObject`, and the mock mirrors that order. In every error case both files are left untouched.

Every operation checks its context before doing any work and returns `ctx.Err()` when it is already done.

## Running the tests

```bash
go test ./... -race -cover
```

`filestore_test.go` holds a contract suite that runs the same scenarios (success, missing source, destination conflict, self-copy, cancelled context) against both providers. `mock` and `s3` also have their own tests for the same error paths, so each package meets CI's 90% per-module coverage minimum on its own. `filestore/cmd/filestore` factors its logic into a `run()` helper that stores, copies and reads a file back; the `main` wrapper itself is intentionally left untested, since it calls `log.Fatal`.
