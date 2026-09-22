# File Store Capability Challenge

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
