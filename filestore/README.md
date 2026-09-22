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

## Storage contract

`FileProvider` exposes context-first `Get`, `Set`, `Purge`, `Copy`, and `Move`.
`PresignedFileProvider` additionally exposes `GetPresignedURL`. Compare wrapped
errors using `errors.Is`.

Keys are opaque strings. With an empty mock `BasePath`, the key is the filename
unchanged; otherwise it is exactly `BasePath + "/" + filename`. Empty names,
leading and repeated slashes, and dot segments are retained. This is a byte
prefix, not filesystem path resolution. The S3-shaped client passes keys to
the configured adapter unchanged; real services may impose additional limits.

`Get` transfers ownership of its body to the caller. Close it on success and
read failure. The demo preserves both read and close failures with `errors.Join`.
Operations check cancellation on entry. The S3 client forwards context to its
adapter; subsequent reader cancellation depends on that adapter. The in-memory
readers do not become cancellable after `Get`, and `MemoryAPI.PutObject` cannot
interrupt a blocked arbitrary `io.Reader` after its entry check.

### Mock ownership

`mock.Config{Bucket: mock.Bucket{Name: "local"}}` remains valid configuration.
Construction snapshots the map and each non-nil file, so separate clients do
not share writes accidentally. Nil seed entries are absent. Do not mutate
seeds concurrently with construction. For intentional sharing, construct one
pointer with `mock.NewSharedBucket(seed)` and attach each client with
`mock.NewClientFromSharedBucket(shared, basePath)`. A nil shared pointer creates
private state. The existing Config structure and NewClient signature are unchanged.
The shared bucket owns both the map and its mutex. Do not copy it after use.
Copy and Move are atomic relative to clients sharing that bucket. Every Get
returns a fresh reader, independent of other reads and the stored object.

### S3-shaped limitations

`s3.ObjectAPI` is an adapter boundary, not an included AWS implementation.
`s3test.MemoryAPI` is the local test adapter; it performs no AWS calls. A nil
`s3.Config.API` reads as an empty bucket, rejects Set with `s3.ErrNoAPI`, makes
Purge a no-op, and reports missing sources for transfers. Adapters translate
missing-object errors to `s3.ErrNoSuchKey`.

Copy checks destination existence, then calls CopyObject. These calls are not
atomic: another writer can create or replace the destination in between and
be overwritten. A client-local mutex cannot coordinate independent clients
or external writers. Move copies and then deletes the source; a failed delete
can leave both objects and returns the adapter's wrapped error. Without
version-aware deletion, a concurrent source replacement can also be removed.

After a failed Move, inspect both objects and coordinate recovery with their
writers using version or conditional-operation support in a real adapter.
Do not blindly delete the destination: it may now contain another writer's
data. This API cannot provide safe automatic rollback.

### Errors and metadata

`ErrNotFound` covers missing reads and transfer sources. Purge is idempotent.
`ErrFileExists` covers detected destination conflicts, including an occupied
key copied or moved onto itself; destination conflict takes precedence over
a missing source. Mock conflict/missing-source errors leave its objects
unchanged. S3 has the concurrency and partial-Move limitations above; arbitrary
adapter errors do not imply that both objects are untouched.

The mock and `s3test.MemoryAPI` use `application/octet-stream` when content type
is empty; explicit values are preserved through Set, Get, Copy, and Move.
Third-party adapters define their own metadata defaults. Mock and in-memory
S3 presigned URLs are illustrative strings and are not served or access grants.

## Run and verify

From this module directory:

    go run ./cmd/filestore
    go test ./... -race -cover
    go vet ./...

From the repository root:

    make test-filestore
    make race-filestore
    make check-filestore
    make cover-filestore
    make docker-check-filestore

See the root README for prerequisites and shared coverage enforcement. The
coverage gate uses the module's aggregate profile, not each package separately.
The contract suite tests both providers; focused tests exercise body closure,
opaque keys, seed snapshots, shared-client races, metadata, and S3 partial
failure/interleaving behavior. The CLI's `run` helper is tested; `main` contains
the logging/exit wrapper. All production and test functions follow the project
complexity limits. Real AWS behavior and throughput are not established by
these in-memory tests.
