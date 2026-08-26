# Project Overview

## What This Is

A Terraform provider for [Apache Superset](https://superset.apache.org/). It lets users manage
Superset resources — databases, datasets, dashboards, charts, roles, users, RLS filters, and CSS
templates — through Terraform HCL.

Built with the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
(not the older SDK).

## Go Module

```text
module terraform-provider-superset
go 1.24
```

The module name is a bare name (no hostname prefix). All internal imports use it as-is:

```go
import "terraform-provider-superset/internal/client"
import "terraform-provider-superset/internal/provider"
```

## Directory Structure

```text
terraform-provider-superset/
├── main.go                          # Entry point — provider address, version injection, debug flag
├── go.mod / go.sum
├── Makefile                         # build, testacc, compose-* targets
├── tools/tools.go                   # Blank import of tfplugindocs to keep it in go.sum
├── terraform-registry-manifest.json # Required for Terraform Registry protocol
│
├── internal/
│   ├── client/                      # Superset HTTP API client
│   │   ├── superset.go              # Auth, CSRF, all CRUD methods (~700 lines)
│   │   ├── superset_test.go
│   │   ├── css_template.go          # CSS template methods (has token-refresh retry logic)
│   │   └── css_template_test.go
│   └── provider/                    # All resources and data sources
│       ├── provider.go              # Wires everything: registers resources + data sources
│       ├── provider_test.go         # Shared test setup (providerConfig, factories)
│       ├── import_helpers.go        # Shared ZIP/hash/skip-pattern helpers for import resources
│       └── <name>_resource.go       # One file per resource
│       └── <name>_data_source.go    # One file per data source
│       └── <name>_resource_test.go  # Acceptance tests alongside the implementation
│
├── docs/                            # Generated — do not edit by hand
│   ├── index.md
│   ├── resources/
│   └── data-sources/
│
├── examples/                        # HCL usage examples (consumed by tfplugindocs)
│   ├── provider/provider.tf
│   ├── resources/<resource_name>/resource.tf
│   └── data-sources/<datasource_name>/data-source.tf
│
└── .github/workflows/
    ├── ci.yml                       # Build, lint, generate-check, tests
    └── release.yml                  # goreleaser + GPG signing on v* tags
```

## Registered Resources (12)

| Terraform type | File |
|---|---|
| `superset_role` | `role_resource.go` |
| `superset_role_permissions` | `role_permissions_resource.go` |
| `superset_database` | `databases_resource.go` |
| `superset_meta_database` | `meta_database_resource.go` |
| `superset_dataset` | `dataset_resource.go` |
| `superset_user` | `user_resource.go` |
| `superset_row_level_security` | `row_level_security_resource.go` |
| `superset_dashboard_import` | `dashboard_import_resource.go` |
| `superset_dashboard_embedding` | `dashboard_embedding_resource.go` |
| `superset_dataset_import` | `dataset_import_resource.go` |
| `superset_chart_import` | `chart_import_resource.go` |
| `superset_css_template` | `css_template_resource.go` |

## Registered Data Sources (7)

| Terraform type | File |
|---|---|
| `superset_role` | `role_data_source.go` |
| `superset_roles` | `roles_data_source.go` |
| `superset_role_permissions` | `role_permissions_data_source.go` |
| `superset_databases` | `databases_data_source.go` |
| `superset_datasets` | `datasets_data_source.go` |
| `superset_users` | `users_data_source.go` |
| `superset_css_template` | `css_template_data_source.go` |

## Provider Configuration

| Attribute | Env var | Default |
|---|---|---|
| `host` | `SUPERSET_HOST` | required |
| `username` | `SUPERSET_USERNAME` | required |
| `password` | `SUPERSET_PASSWORD` | required |
| `provider` | `SUPERSET_PROVIDER` | `"db"` |

## Key Dependencies

| Package | Purpose |
|---|---|
| `terraform-plugin-framework v1.16.1` | All resources and data sources |
| `terraform-plugin-log v0.9.0` | Structured logging via `tflog` |
| `terraform-plugin-testing v1.13.3` | Acceptance test framework |
| `jarcoal/httpmock v1.4.1` | HTTP mocking in tests |
| `stretchr/testify v1.10.0` | Test assertions |
| `gopkg.in/yaml.v3` | YAML parsing in import resources |

## Known Rough Edges

- Several `fmt.Printf("DEBUG ...")` calls exist in `internal/client/superset.go`
(in `GetAllDatabases`, `GetMetaDatabase`, `CreateDataset`, `UpdateDataset`).
These should be replaced with `tflog.Debug` when touched.
- `errcheck` is disabled in golangci-lint.
Many error returns from `io.ReadAll` and `resp.Body.Close()` are intentionally ignored.
- The global database cache in `superset.go` is process-scoped with a 5-minute TTL (`globalDatabasesCache`).
Call `ClearGlobalDatabaseCache()` in tests that exercise database lookups.
