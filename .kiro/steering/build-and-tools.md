# Build and Tools

## Makefile Targets

```shell
make build     # Build binary to dist/terraform-provider-superset
make testacc   # Run acceptance tests (TF_ACC=1 go test ./... -timeout 120m)
```

Pass additional Go test flags via `TESTARGS`:

```shell
make testacc TESTARGS="-run TestAccRoleResource -count=1"
```

## Building

```shell
# Build to dist/ (via Makefile)
make build

# Install to $GOPATH/bin directly
go install .

# Build manually with version injection
go build -ldflags="-X main.version=v1.2.3" -o dist/terraform-provider-superset .
```

The `dist/` directory is in `.gitignore`.

## Linting

This project uses `golangci-lint` v2.

```shell
# Run lint (same as CI)
golangci-lint run

# Verify config is valid
golangci-lint config verify
```

**Enabled linters:** `durationcheck`, `forcetypeassert`, `godot`, `ineffassign`, `makezero`, `misspell`,
`nilerr`, `predeclared`, `staticcheck`, `unconvert`, `unparam`, `unused`, `govet`, `copyloopvar`, `usetesting`

**Disabled:** `errcheck` — many `io.ReadAll` and `resp.Body.Close()` errors are intentionally ignored.

**Formatter:** `gofmt` (enforced by golangci-lint, not a separate step).

Notable rules to watch:

- `godot` — all exported comments must end with a period.
- `forcetypeassert` — prefer comma-ok type assertions over bare ones (except where the existing code
  pattern already uses bare assertions in response parsing).
- `usetesting` — use `t.Setenv` / `t.TempDir` instead of `os.Setenv` / `os.MkdirTemp` in tests.

## Generating Documentation

```shell
go generate ./...
```

This runs two generators defined in `main.go`:

1. `terraform fmt -recursive ./examples/` — formats all example `.tf` files in place.
2. `tfplugindocs generate -provider-name superset` — regenerates everything under `docs/` from schema
   descriptions and examples.

**Never edit `docs/` by hand** — it is fully generated. Edit schema `Description` fields and example
files under `examples/` instead.

The CI `generate` job runs `go generate ./...` and then checks `git diff --exit-code` to enforce that
generated docs are committed. Run `go generate ./...` and commit the output before pushing.

## Pre-commit Hooks

```shell
# Install hooks
pre-commit install

# Run all hooks manually
pre-commit run --all-files
```

Hooks configured in `.pre-commit-config.yaml`:

| Hook | What it checks/fixes |
|---|---|
| `check-merge-conflict` | No unresolved merge markers |
| `check-added-large-files` | No accidental large file commits |
| `detect-private-key` | No private keys committed |
| `trailing-whitespace` | Trims trailing whitespace |
| `end-of-file-fixer` | Ensures files end with a newline |
| `prettier` | Formats `.yml`/`.yaml`/`.avsc` files |
| `markdownlint` | Lints and auto-fixes Markdown (excludes `docs/`) |
| `golangci-lint-full` | Full lint pass |
| `golangci-lint-config-verify` | Validates `.golangci.yml` |

## License Headers

The `.copywrite.hcl` file configures HashiCorp's `copywrite` tool to add MPL-2.0 license headers to
Go files. Excluded paths: `examples/**`, `.github/ISSUE_TEMPLATE/*.yml`, `.golangci.yml`,
`.goreleaser.yml`.

When adding new Go files, the CI `prek` job will flag missing headers. Add the header manually if not
using the `copywrite` tool:

```go
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
```

## CI Pipeline (`.github/workflows/ci.yml`)

Triggered on push to `main` and all PRs (ignores README-only changes).

| Job | What it does |
|---|---|
| `prek` | Runs `j178/prek-action` (pre-commit checks) |
| `build` | `go build -v .` + golangci-lint v2.8.0 |
| `generate` | `go generate ./...` + `git diff --exit-code` |
| `test` | Acceptance tests with httpmock (no real Superset needed) |

The `test` job sets these env vars so the provider can initialize:

```text
TF_ACC=1
SUPERSET_USERNAME=fake-username
SUPERSET_PASSWORD=fake-password
SUPERSET_HOST=http://superset-host
```

Go version is resolved from `go.mod` via `go-version-file: "go.mod"` in all jobs.

## Release Pipeline (`.github/workflows/release.yml`)

Triggered on `v*` tags. Uses goreleaser.

```shell
# Create a release
git tag v1.2.3
git push origin v1.2.3
```

Goreleaser builds for: `linux`, `darwin`, `windows`, `freebsd` × `amd64`, `386`, `arm`, `arm64`
(darwin skips `386`). Outputs ZIP archives + SHA256 checksums + GPG signature.

Required repository secrets: `GPG_PRIVATE_KEY`, `PASSPHRASE`, `GITHUB_TOKEN`.

Binary name pattern: `terraform-provider-superset_v<version>`.
Archive name: `terraform-provider-superset_<version>_<os>_<arch>.zip`.

## go.mod and Dependencies

```shell
# Add a dependency
go get github.com/author/package@v1.2.3
go mod tidy

# Update all dependencies
go get -u ./...
go mod tidy
```

Always commit both `go.mod` and `go.sum`. The `tools/tools.go` file contains a blank import of
`tfplugindocs` — do not remove it, it keeps the tool in `go.sum` for `go generate`.
