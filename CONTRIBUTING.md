# Contributing to Pando

Thanks for helping improve Pando. Small, focused changes are easiest to review.

## Development setup

Install Go 1.27.1 or newer and, optionally, `mise`. Clone the repository and
run the test suite:

```sh
go test ./...
go vet ./...
```

Format Go changes with `gofmt` before opening a pull request. The multi-node
local task starts four Pando processes on ports 8080, 8081, 8082, and 8083,
using temporary backup directories under `/tmp`:

```sh
mise run test
```

The task builds `/tmp/pando` first and gives the processes local `SELF_URL`
and `BOOTSTRAP_PEER_URL` values. It does not provide an Immich server, so use it
for peer and startup behavior rather than a complete backup run.

## Project layout

- `internal/backup`: small wrapper around Immich bulk downloads.
- `internal/config`: environment configuration and parsing.
- `internal/health`: Immich, API-key, storage, and system health checks.
- `internal/immich`: Immich API access, asset polling, and downloads.
- `internal/peers`: peer identity, registration, health, and relay behavior.
- `internal/server`: HTTP routes, handlers, and the embedded dashboard.
- `internal/service`: scan orchestration, placement, manifest, and readiness logic.
- `internal/status`: service state and operational counters.
- `internal/utils`: shared file, HTTP, and utility helpers.

## Pull requests

Keep pull requests small and explain the behavior you changed. Include the
commands you ran, such as `go test ./...` and `go vet ./...`, and call out any
checks you could not run. Update the README when configuration, routes, or
operator behavior changes.

## Bug reports

Include the Pando commit or version, Go version, operating system, deployment
method, relevant configuration shape, and the logs around the failure. Remove
`IMMICH_API_TOKEN` and any other secrets from logs before sharing them. A small
reproduction or the exact request and response is useful when reporting an HTTP
or peer-registration problem.
