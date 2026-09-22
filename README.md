# go-challenge

This repo is organized into multiple challenge exercises, each within its own subdirectory.

Please follow the specific instructions for the challenge you have been assigned, and then submit your changes for code review based on the guidelines below.

## Modules

- [carrierproxy](carrierproxy/README.md): browser-backed policy provider.
- [comms](comms/README.md): HTTP API for communication operations.
- [filestore](filestore/README.md): provider-neutral object storage contracts and adapters.
- [notifier](notifier/README.md): topic-to-email notification producer.

Each directory is an independent Go module. Run module commands through the
root Makefile; `go test ./...` from the repository root is not supported because
the root is intentionally not a Go module or workspace.

## Prerequisites

- Go version declared by each module's `go.mod`.
- golangci-lint v2.13.2 for local lint targets.
- GNU Make.
- Docker only when using `docker-*` targets.
- A locally installed Chrome/Chromium for carrierproxy's real-browser tests.
  Browser-dependent tests skip when it is absent, so a successful test command
  alone does not establish browser integration coverage.

## Verification

Run from the repository root:

```bash
make check
make cover
```

`make check` runs formatting, tidy, build, vet, default lint, complexity lint,
race tests, and the 90% per-module coverage gate. Use
`make check-<module>` for one module or `make docker-check` for the pinned Docker
toolchain. Complexity limits apply to production and test functions: cyclomatic
complexity at most 6 and cognitive complexity at most 10.

The coverage gate checks the aggregate statement coverage reported by
`go tool cover` for each module, not an average of package percentages. Local
checks and CI call the same `scripts/check-coverage.sh` helper through the
Makefile. `make coverage-check` reruns just the race-enabled coverage gate;
`make coverage-check-filestore` checks one module.

Carrierproxy's coverage check supplies placeholder credentials (`ci-user` and
`ci-pass`) to a local fixture site configured to accept those values. They are
not real secrets and do not access an external account. Chrome/Chromium must
also be available in the environment running the tests, including inside a
Docker container. The toolchain image is pinned to
`golangci/golangci-lint:v2.13.2` and does not include Chrome/Chromium. Without
a browser, the carrierproxy coverage gate can fail after its browser tests
skip; the default Docker image alone is not sufficient for full verification.

`make cover` generates `.cache/coverage/<module>.html` and the corresponding
`.out` profiles. These reports are ignored by Git. This report-generation
target does not enforce the threshold or enable the credential-gated
carrierproxy login test itself. To include that fixture test in the reports:

```bash
CARRIERPROXY_USERNAME=ci-user CARRIERPROXY_PASSWORD=ci-pass make cover
```

Use `make help` for all targets. Common focused checks are:

```bash
make test-filestore
make complexity
make docker-lint-comms
```

These checks exercise local fixtures and bundled stand-ins. They do not prove
delivery through external email services, hosted S3 behavior, or deployment
readiness; see each module's README for its contracts and limitations.

## General Submission Guidelines

* [ ] Fork this repository and make your changes directly within the appropriate challenge directory.
* [ ] When complete, send in a pull request for code review.
* [ ] Ensure that before you send the PR that the main application compiles and runs, the tests all pass, and your commits are all squashed into a single commit.
