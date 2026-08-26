# Contributing to terraform-provider-superset

Thanks for taking the time to contribute. This guide covers everything you need to go from zero to
a merged pull request.

## Table of Contents

- [Contributing to terraform-provider-superset](#contributing-to-terraform-provider-superset)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Setting Up the Development Environment](#setting-up-the-development-environment)
    - [Using a local build with Terraform](#using-a-local-build-with-terraform)
  - [Project Structure](#project-structure)
  - [Building](#building)
  - [Running Tests](#running-tests)
    - [Writing tests](#writing-tests)
  - [Linting and Code Style](#linting-and-code-style)
  - [Generating Documentation](#generating-documentation)
  - [Adding a New Resource or Data Source](#adding-a-new-resource-or-data-source)
    - [License headers](#license-headers)
  - [Commit and PR Guidelines](#commit-and-pr-guidelines)
  - [CI Pipeline](#ci-pipeline)
  - [License](#license)

---

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | >= 1.24 | <https://golang.org/doc/install> |
| Terraform or OpenTofu | >= 1.0 | <https://developer.hashicorp.com/terraform/downloads> |
| golangci-lint | v2.x | <https://golangci-lint.run/welcome/install/> |
| pre-commit | any | <https://pre-commit.com/#install> |
| tfplugindocs | latest | installed automatically via `go generate` |

---

## Setting Up the Development Environment

```shell
# Clone the repository
git clone https://github.com/gr8-toolkit/terraform-provider-superset.git
cd terraform-provider-superset

# Download Go module dependencies
go mod download

# Install pre-commit hooks (runs linters and formatters on every commit)
pre-commit install
```

### Using a local build with Terraform

To test a locally built provider binary with Terraform, create a
[development override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
in your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "gr8-toolkit/superset" = "/path/to/terraform-provider-superset/dist"
  }
  direct {}
}
```

Then build the provider and run `terraform plan` in any directory that uses it:

```shell
make build
cd /your/terraform/config
terraform plan
```

---

## Project Structure

```text
terraform-provider-superset/
├── main.go                    # Entry point
├── internal/
│   ├── client/                # Superset HTTP API client (auth, CSRF, CRUD)
│   │   ├── superset.go
│   │   └── *_test.go
│   └── provider/              # Terraform resources and data sources
│       ├── provider.go        # Registers all resources and data sources
│       ├── provider_test.go   # Shared test setup (providerConfig, factories)
│       ├── import_helpers.go  # ZIP/hash helpers for import-type resources
│       └── <name>_resource.go / <name>_data_source.go
├── examples/                  # HCL usage examples (consumed by tfplugindocs)
└── docs/                      # Generated — do not edit by hand
```

---

## Building

```shell
# Build binary to dist/
make build

# Or install directly to $GOPATH/bin
go install .
```

---

## Running Tests

All provider tests in `internal/provider/` are acceptance tests that use
[httpmock](https://github.com/jarcoal/httpmock) to intercept HTTP calls. No real Superset
instance is required.

```shell
# Run all acceptance tests
make testacc

# Run a specific test by name
make testacc TESTARGS="-run TestAccRoleResource"

# Run client unit tests (no TF_ACC needed)
go test ./internal/client/ -v
```

The `TF_ACC=1` environment variable is set automatically by `make testacc`. The provider is
configured to connect to `http://superset-host` with fake credentials — every HTTP call is
intercepted by httpmock.

### Writing tests

Every new resource or data source must include acceptance tests. Follow the pattern used by
existing tests (e.g. `css_template_resource_test.go`):

1. Activate httpmock and register responders for login, CSRF, and all resource endpoints.
2. Use `resource.Test` with at least a Create+Read step, an Update step (if the resource supports
   update), and an `ImportState` step.
3. Place test files alongside the implementation: `internal/provider/<name>_resource_test.go`.

See the [testing rules](.kiro/steering/testing.md) for the full pattern and httpmock reference.

---

## Linting and Code Style

```shell
golangci-lint run
```

The project enforces `gofmt` formatting plus a set of linters defined in `.golangci.yml`.
Notable rules:

- All exported symbols must have a doc comment ending with a period (`godot`).
- Prefer comma-ok type assertions over bare ones (`forcetypeassert`).
- Use `t.Setenv` / `t.TempDir` in tests instead of `os.Setenv` / `os.MkdirTemp` (`usetesting`).
- `errcheck` is disabled — `io.ReadAll` and `resp.Body.Close()` errors are intentionally ignored.

Pre-commit runs the full lint pass on every commit. You can also run all hooks manually:

```shell
pre-commit run --all-files
```

---

## Generating Documentation

Documentation under `docs/` is fully generated — never edit it by hand.

```shell
go generate ./...
```

This runs two steps:

1. `terraform fmt -recursive ./examples/` — formats all example `.tf` files.
2. `tfplugindocs generate` — regenerates `docs/` from schema descriptions and `examples/`.

**Commit the generated output.** CI checks that `git diff --exit-code docs/` is clean after
running `go generate ./...`. If you forget to regenerate, the `generate` job will fail.

When adding a new resource, also add a minimal example file:

```text
examples/resources/superset_<name>/resource.tf
examples/data-sources/superset_<name>/data-source.tf   # if adding a data source
```

---

## Adding a New Resource or Data Source

Follow this checklist (replace `foo` / `Foo` with your resource name):

1. **Client methods** — add `CreateFoo`, `GetFoo`, `UpdateFoo`, `DeleteFoo` to
   `internal/client/superset.go`. All mutating calls require a CSRF token; read-only GETs do not.

2. **Resource file** — create `internal/provider/foo_resource.go`. Required sections:
   - Compile-time interface assertions
   - `NewFooResource()` constructor
   - `fooResource` struct (holds `*client.Client`)
   - `fooResourceModel` struct with `tfsdk` tags
   - `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`, `ImportState`

3. **Data source file** (if needed) — create `internal/provider/foos_data_source.go` with the
   same pattern but only a `Read` method and all-`Computed` schema attributes.

4. **Register** — add the constructor to `Resources()` or `DataSources()` in
   `internal/provider/provider.go`.

5. **Example** — add `examples/resources/superset_foo/resource.tf`.

6. **Tests** — add `internal/provider/foo_resource_test.go` covering create, update, import, and
   delete.

7. **Regenerate docs** — run `go generate ./...` and commit the output.

8. **Update README** — add a row to the Resources or Data Sources tables.

See the [coding conventions](.kiro/steering/coding-conventions.md) for detailed patterns, and
[new-resource-checklist](.kiro/steering/new-resource-checklist.md) for the full step-by-step guide.

### License headers

All new Go files must include the MPL-2.0 header at the top:

```go
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
```

CI will fail without it.

---

## Commit and PR Guidelines

- Follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`,
  `chore:`, `test:`, `refactor:`.
- Keep commits focused. One logical change per commit.
- Open a pull request against `main`. The PR title must follow Conventional Commits (enforced by
  the `semantic.yml` config).
- Every PR must pass all CI jobs before merging: `prek`, `build`, `generate`, and `test`.
- Add or update tests for any behaviour change.
- Regenerate docs (`go generate ./...`) and commit the result if schema or examples changed.

---

## CI Pipeline

| Job | What it does |
|-----|--------------|
| `prek` | Runs pre-commit hooks (formatting, lint, secret detection) |
| `build` | Compiles the provider and runs golangci-lint |
| `generate` | Runs `go generate ./...` and checks for uncommitted diffs |
| `test` | Runs acceptance tests with httpmock against Terraform 1.8 |

All jobs are defined in `.github/workflows/ci.yml` and run on every push to `main` and on every
pull request (excluding README-only changes).

---

## License

By contributing, you agree that your contributions will be licensed under the
[Mozilla Public License 2.0](LICENSE).
