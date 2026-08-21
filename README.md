# Terraform Provider for Apache Superset

A Terraform provider for managing [Apache Superset](https://superset.apache.org/) resources — databases, datasets, dashboards, charts, roles, users, and more.

## Requirements

| Tool | Version |
|------|---------|
| [Terraform](https://developer.hashicorp.com/terraform/downloads) | >= 1.0 |
| [Go](https://golang.org/doc/install) | >= 1.24 |

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
  url      = "https://your-superset-instance.example.com"
  username = "admin"
  password = "your-password"
}
```

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
| `superset_css_template` | Look up a CSS template |

## Building the Provider

Clone the repository and build the binary to `dist/`:

```shell
git clone https://github.com/gr8-toolkit/terraform-provider-superset.git
cd terraform-provider-superset
make build
```

The binary will be placed at `dist/terraform-provider-superset`.

To install it into `$GOPATH/bin` directly:

```shell
go install .
```

## Development

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).

```shell
go get github.com/author/dependency
go mod tidy
```

Commit the resulting changes to `go.mod` and `go.sum`.

### Generating Documentation

```shell
go generate
```

This formats the example Terraform files and regenerates the `docs/` directory using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs).

### Running Acceptance Tests

> **Note:** Acceptance tests create real resources against a running Superset instance.

```shell
make testacc
```

## License

[Mozilla Public License 2.0](LICENSE)
