# Terraform Provider for Apache Superset

[![CI](https://github.com/gr8-toolkit/terraform-provider-superset/actions/workflows/ci.yml/badge.svg)](https://github.com/gr8-toolkit/terraform-provider-superset/actions/workflows/ci.yml)
[![prek](https://img.shields.io/badge/hooks-prek-blue)](https://github.com/j178/prek)
[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-gr8--toolkit%2Fsuperset-7B42BC?logo=terraform)](https://registry.terraform.io/providers/gr8-toolkit/superset/latest)
[![OpenTofu Registry](https://img.shields.io/badge/OpenTofu%20Registry-gr8--toolkit%2Fsuperset-FFDA18?logo=opentofu&logoColor=000)](https://search.opentofu.org/provider/gr8-toolkit/superset/latest)

A Terraform/OpenTofu provider for managing [Apache Superset](https://superset.apache.org/)
resources declaratively — databases, datasets, dashboards, charts, roles, users,
row-level security filters, saved queries, annotation layers, and more.

Superset is an open-source business intelligence platform. This provider lets you
version-control and automate your Superset configuration the same way you manage
the rest of your infrastructure.

## Requirements

| Tool | Minimum version |
|------|----------------|
| [Terraform](https://developer.hashicorp.com/terraform/downloads) or [OpenTofu](https://opentofu.org/docs/intro/install/) | 1.0 |
| [Go](https://golang.org/doc/install) | 1.24 (only for building from source) |
| [Docker + Compose v2](https://docs.docker.com/compose/) | any recent (only for running acceptance tests) |

## Using the Provider

```hcl
terraform {
  required_providers {
    superset = {
      source = "gr8-toolkit/superset"
    }
  }
}

provider "superset" {
  host     = "https://your-superset-instance.example.com"
  username = "admin"
  password = "your-password"
}
```

Provider configuration can also come from environment variables:

| Attribute  | Environment variable | Default    |
|------------|----------------------|------------|
| `host`     | `SUPERSET_HOST`      | (required) |
| `username` | `SUPERSET_USERNAME`  | (required) |
| `password` | `SUPERSET_PASSWORD`  | (required) |
| `provider` | `SUPERSET_PROVIDER`  | `"db"`     |

## Resources

| Resource | Description |
|----------|-------------|
| `superset_chart` | Manage charts (native CRUD) |
| `superset_chart_import` | Import charts from an export ZIP directory |
| `superset_css_template` | Manage CSS templates |
| `superset_dashboard` | Manage dashboards (native CRUD) |
| `superset_dashboard_embedding` | Configure dashboard embedding settings |
| `superset_dashboard_import` | Import dashboards from an export ZIP directory |
| `superset_database` | Manage database connections |
| `superset_dataset` | Manage datasets |
| `superset_dataset_import` | Import datasets from an export ZIP directory |
| `superset_meta_database` | Manage meta (cross-database query) connections |
| `superset_role` | Manage roles |
| `superset_role_permissions` | Manage permission assignments for a role |
| `superset_row_level_security` | Manage row-level security filters |
| `superset_saved_query` | Manage saved SQL queries |
| `superset_annotation_layer` | Manage annotation layers |
| `superset_user` | Manage users |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `superset_css_template` | Look up a CSS template by name |
| `superset_databases` | List all database connections |
| `superset_datasets` | List all datasets |
| `superset_role` | Look up a role by name |
| `superset_roles` | List all roles |
| `superset_role_permissions` | List permissions assigned to a role |
| `superset_users` | List all users |

## Examples

```hcl
# Create a database connection
resource "superset_database" "warehouse" {
  connection_name  = "Data Warehouse"
  db_engine        = "postgresql"
  db_user          = "reader"
  db_pass          = var.db_password
  db_host          = "warehouse.internal"
  db_port          = 5432
  db_name          = "analytics"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_run_async  = true
  expose_in_sqllab = true
}

# Create a dataset backed by that database
resource "superset_dataset" "orders" {
  table_name    = "orders"
  database_name = superset_database.warehouse.connection_name
  schema        = "public"
}

# Create a chart
resource "superset_chart" "orders_by_day" {
  slice_name    = "Orders by day"
  viz_type      = "echarts_timeseries_bar"
  datasource_id = superset_dataset.orders.id
  params        = jsonencode({
    metrics    = ["count"]
    time_grain = "P1D"
  })
}

# Create a dashboard
resource "superset_dashboard" "ops" {
  dashboard_title = "Operations"
  published       = true
}

# Create a role and assign permissions
resource "superset_role" "analyst" {
  name = "Analyst"
}

resource "superset_role_permissions" "analyst" {
  role_name = superset_role.analyst.name
  resource_permissions = [
    { permission = "can_read", view_menu = "Chart" },
    { permission = "can_read", view_menu = "Dashboard" },
  ]
}

# Saved query
resource "superset_saved_query" "daily_orders" {
  database_id = superset_database.warehouse.id
  label       = "Daily order count"
  sql         = "SELECT DATE(created_at), COUNT(*) FROM orders GROUP BY 1"
  schema      = "public"
}

# Annotation layer
resource "superset_annotation_layer" "deployments" {
  name        = "Deployments"
  description = "Production deployment events"
}
```

## Building the Provider

```shell
git clone https://github.com/gr8-toolkit/terraform-provider-superset.git
cd terraform-provider-superset

# Build to dist/
make build

# Install directly into $GOPATH/bin
go install .
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for a full guide on setting up the
development environment, writing new resources, and submitting pull requests.

### Available make targets

```shell
make help          # List all targets
make build         # Build provider binary to dist/
make generate      # Regenerate docs from schema descriptions
make lint          # Run golangci-lint

# Tests
make testacc       # Acceptance tests (starts docker-compose automatically)
make test-client   # Client unit tests (no Superset needed)

# Docker Compose
make compose-up    # Start the local Superset stack
make compose-down  # Stop and remove the stack (volumes included)
make compose-logs  # Tail container logs
make compose-ps    # Show container status
make compose-restart  # Restart the Superset web container
```

### Running tests

#### Client unit tests — fast, no Superset needed

The `internal/client/` package is tested with `httpmock`. These run in under a second:

```shell
make test-client
```

#### Acceptance tests — run against a real Superset

Acceptance tests require Docker and Docker Compose v2.

```shell
# Test against Superset 6.1.0 (default)
make testacc

# Test against Superset 6.0.0
make testacc SUPERSET_VERSION=6.0.0

# Filter to a specific test
make testacc TESTARGS="-run TestAccRoleResource"
```

`make testacc` starts the compose stack, waits for Superset to be healthy,
runs all tests, then tears the stack down.

#### Running tests without restarting Superset

Useful when iterating on a single test:

```shell
# Start Superset once (admin/admin on localhost:8088)
make compose-up

# Run tests as many times as you like
TF_ACC=1 SUPERSET_HOST=http://localhost:8088 \
  SUPERSET_USERNAME=admin SUPERSET_PASSWORD=admin \
  go test ./internal/provider/ -v -run TestAccRoleResource -timeout 10m

# Tear down when done
make compose-down
```

#### Supported Superset versions

CI runs the full test suite against both versions on every push:

| Version |
|---------|
| 6.0.0   |
| 6.1.0   |

## Repository layout

```text
internal/
  client/       # Superset HTTP API client, one file per resource group
  provider/     # Terraform resources and data sources
docker-compose/ # Compose stack for acceptance tests
scripts/        # run-acc-tests.sh helper
examples/       # HCL usage examples (consumed by tfplugindocs)
docs/           # Generated documentation — do not edit by hand
```

## License

[Mozilla Public License 2.0](LICENSE)
