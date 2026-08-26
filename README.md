# Terraform Provider for Apache Superset

[![CI](https://github.com/gr8-toolkit/terraform-provider-superset/actions/workflows/ci.yml/badge.svg)](https://github.com/gr8-toolkit/terraform-provider-superset/actions/workflows/ci.yml)
[![prek](https://img.shields.io/badge/hooks-prek-blue)](https://github.com/j178/prek)
[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-gr8--toolkit%2Fsuperset-7B42BC?logo=terraform)](https://registry.terraform.io/providers/gr8-toolkit/superset/latest)
[![OpenTofu Registry](https://img.shields.io/badge/OpenTofu%20Registry-gr8--toolkit%2Fsuperset-FFDA18?logo=opentofu&logoColor=000)](https://search.opentofu.org/provider/gr8-toolkit/superset/latest)

A Terraform/OpenTofu provider for managing [Apache Superset](https://superset.apache.org/)
resources declaratively — databases, datasets, dashboards, charts, roles, users, row-level
security filters, and more.

Superset is an open-source business intelligence platform. This provider lets you version-control
and automate your Superset configuration the same way you manage the rest of your infrastructure.

## Requirements

| Tool | Version |
|------|---------|
| [Terraform](https://developer.hashicorp.com/terraform/downloads) or [OpenTofu](https://opentofu.org/docs/intro/install/) | >= 1.0 |
| [Go](https://golang.org/doc/install) | >= 1.24 (only for building from source) |

## Using the Provider

```hcl
terraform {
  required_providers {
    superset = {
      source  = "gr8-toolkit/superset"
    }
  }
}

provider "superset" {
  host     = "https://your-superset-instance.example.com"
  username = "admin"
  password = "your-password"
}
```

Provider configuration can also be supplied via environment variables:

| Attribute  | Environment Variable  | Default    |
|------------|-----------------------|------------|
| `host`     | `SUPERSET_HOST`       | (required) |
| `username` | `SUPERSET_USERNAME`   | (required) |
| `password` | `SUPERSET_PASSWORD`   | (required) |
| `provider` | `SUPERSET_PROVIDER`   | `"db"`     |

## Resources

| Resource | Description |
|----------|-------------|
| `superset_database` | Manage database connections |
| `superset_dataset` | Manage datasets |
| `superset_dataset_import` | Import datasets from a ZIP file |
| `superset_chart_import` | Import charts from a ZIP file |
| `superset_dashboard_import` | Import dashboards from a ZIP file |
| `superset_dashboard_embedding` | Configure dashboard embedding settings |
| `superset_meta_database` | Manage meta (virtual) databases |
| `superset_role` | Manage roles |
| `superset_role_permissions` | Manage role permission assignments |
| `superset_row_level_security` | Manage row-level security filters |
| `superset_user` | Manage users |
| `superset_css_template` | Manage CSS templates |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `superset_databases` | List all database connections |
| `superset_datasets` | List all datasets |
| `superset_role` | Look up a role by name |
| `superset_roles` | List all roles |
| `superset_role_permissions` | List permissions for a role |
| `superset_users` | List all users |
| `superset_css_template` | Look up a CSS template by name |

## Building the Provider

Clone the repository and build the binary to `dist/`:

```shell
git clone https://github.com/gr8-toolkit/terraform-provider-superset.git
cd terraform-provider-superset
make build
```

The binary will be placed at `dist/terraform-provider-superset`.

To install it directly into `$GOPATH/bin`:

```shell
go install .
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for a full guide on setting up the development environment,
running tests, writing new resources, and submitting pull requests.

### Quick start

```shell
# Build
make build

# Regenerate docs
make generate

# Lint
make lint

# Show all available targets
make help
```

### Running acceptance tests

Acceptance tests run against a **real Superset instance** via Docker Compose.
You need Docker and Docker Compose v2 installed.

#### One-command test run (recommended)

```shell
# Test against Superset 6.1.0 (default)
make testacc

# Test against Superset 6.0.0
make testacc SUPERSET_VERSION=6.0.0

# Filter to a specific test
make testacc TESTARGS="-run TestAccRoleResource"

# Both at once
make testacc SUPERSET_VERSION=6.0.0 TESTARGS="-run TestAccRoleResource -count=1"
```

`make testacc` starts the compose stack, waits for Superset to be healthy, runs
the tests, then tears the stack down automatically.

#### Managing the stack manually

Useful when you want to run tests repeatedly without restarting Superset each time:

```shell
# Start Superset 6.1.0 on localhost:8088 (admin/admin)
make compose-up

# (optional) tail logs
make compose-logs

# Run tests against the running instance
TF_ACC=1 SUPERSET_HOST=http://localhost:8088 \
  SUPERSET_USERNAME=admin SUPERSET_PASSWORD=admin \
  go test ./internal/provider/ -v -timeout 30m

# Or filter
TF_ACC=1 SUPERSET_HOST=http://localhost:8088 \
  SUPERSET_USERNAME=admin SUPERSET_PASSWORD=admin \
  go test ./internal/provider/ -v -run TestAccRoleResource -timeout 10m

# Stop and clean up when done
make compose-down

# Use a different version
make compose-up SUPERSET_VERSION=6.0.0
make compose-down SUPERSET_VERSION=6.0.0
```

#### Client unit tests

The `internal/client/` package has fast unit tests that use `httpmock` and
need no running Superset:

```shell
make test-client
```

#### Supported Superset versions

Tests are verified against the following versions in CI:

| Version |
|---------|
| 6.0.0   |
| 6.1.0   |

## License

[Mozilla Public License 2.0](LICENSE)
