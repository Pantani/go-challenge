# go-challenge

This repository contains four independent Go interview exercises:
[carrierproxy](carrierproxy/README.md), [comms](comms/README.md),
[filestore](filestore/README.md), and [notifier](notifier/README.md).

Each directory is its own Go module. Run a focused check from a module:

```sh
cd filestore
go test -race ./...
go vet ./...
```

From the repository root, `make check` runs formatting, module tidiness,
build, vet, lint, complexity, and race-test checks. Carrierproxy's
real-browser test reads `CARRIERPROXY_USERNAME` and `CARRIERPROXY_PASSWORD`
and, by default, logs into a local fixture site, so it needs no external
account; `CARRIERPROXY_LOGIN_URL` points it at a real site instead.

## General Submission Guidelines

* [ ] Fork this repository and make changes in the assigned challenge directory.
* [ ] Submit a pull request for code review.
* [ ] Before submission, ensure the relevant application compiles, tests pass,
  and commits are squashed into one commit.
