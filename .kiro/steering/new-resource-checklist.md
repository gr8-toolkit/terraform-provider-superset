# Adding a New Resource or Data Source

Follow these steps in order. Replace `foo` / `Foo` / `FOO` with your actual resource name.

---

## 1. Add Client Methods (`internal/client/superset.go`)

Add CRUD methods to the client. Follow the conventions in `coding-conventions.md`.

Minimum required methods for a full resource:

- `CreateFoo(...)` — POST, needs CSRF token
- `GetFoo(id int64)` — GET, no CSRF
- `UpdateFoo(id int64, ...)` — PUT/POST, needs CSRF token
- `DeleteFoo(id int64)` — DELETE, needs CSRF token

For a data source (read-only): only `GetFoo` or a list method like `GetAllFoos`.

If you need a structured return type, define it near the top of `superset.go` alongside the existing
types (`Role`, `User`, `MetaDatabase`, etc.):

```go
type Foo struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}
```

---

## 2. Create the Resource File (`internal/provider/foo_resource.go`)

Use the template from `coding-conventions.md`. Required sections:

- [ ] Compile-time interface assertions (`var _ resource.Resource = ...`)
- [ ] `NewFooResource() resource.Resource` constructor
- [ ] `fooResource` struct with `client *client.Client`
- [ ] `fooResourceModel` struct with `tfsdk` tags
- [ ] `Metadata` — sets `TypeName = req.ProviderTypeName + "_foo"`
- [ ] `Schema` — every attribute has a `Description`; ID uses `UseStateForUnknown()`
- [ ] `Configure` — asserts `req.ProviderData.(*client.Client)`
- [ ] `Create` — reads plan, calls client, sets state
- [ ] `Read` — reads state, calls client, updates state; calls `resp.State.RemoveResource` on 404
- [ ] `Update` — reads plan + state, calls client, sets updated state
- [ ] `Delete` — reads state, calls client; handles 404 gracefully
- [ ] `ImportState` — parses string ID to `int64`, sets via `path.Root("id")`

---

## 3. Create the Data Source File (if needed) (`internal/provider/foos_data_source.go`)

- [ ] Compile-time interface assertions (`var _ datasource.DataSource = ...`)
- [ ] `NewFoosDataSource() datasource.DataSource` constructor
- [ ] `foosDataSource` struct with `client *client.Client`
- [ ] Model structs with all-`Computed` schema attributes
- [ ] `Metadata` — sets `TypeName = req.ProviderTypeName + "_foos"`
- [ ] `Schema`
- [ ] `Configure` — same pattern as resources
- [ ] `Read` — calls client list method, maps into state

---

## 4. Register in Provider (`internal/provider/provider.go`)

Add the constructor to the appropriate slice:

```go
// For a resource:
func (p *supersetProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ...existing entries...
        NewFooResource,
    }
}

// For a data source:
func (p *supersetProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ...existing entries...
        NewFoosDataSource,
    }
}
```

---

## 5. Add an Example (`examples/resources/superset_foo/resource.tf`)

Create a minimal but realistic HCL example. This file is consumed by `tfplugindocs` to populate the generated docs.

```hcl
resource "superset_foo" "example" {
  name = "example-foo"
}
```

For a data source: `examples/data-sources/superset_foo/data-source.tf`

```hcl
data "superset_foos" "all" {}
```

---

## 6. Write Tests (`internal/provider/foo_resource_test.go`)

Follow the pattern in `testing.md`. Minimum test coverage:

- [ ] Create — verifies all required attributes in state
- [ ] Read (implicit in every step)
- [ ] Update — at least one attribute change
- [ ] ImportState step (`ImportState: true, ImportStateVerify: true`)
- [ ] Delete (implicit in test cleanup)

Mock every HTTP endpoint the resource touches:

- `POST /api/v1/security/login` (authentication)
- `GET /api/v1/security/csrf_token/` (before every mutating call)
- Resource-specific create/read/update/delete endpoints

---

## 7. Regenerate Docs

```shell
go generate ./...
```

This updates `docs/resources/foo.md` and `docs/data-sources/foo.md` from your schema descriptions
and example files. Commit the generated output.

---

## 8. Update README

Add a row to the Resources or Data Sources tables in `README.md`:

```markdown
| `superset_foo` | Manage foos |
```

---

## 9. Verify

```shell
# Build must succeed
make build

# Lint must pass
golangci-lint run

# Generated docs must be committed
go generate ./...
git diff --exit-code docs/

# Tests must pass
make testacc TESTARGS="-run TestAccFooResource"
```

---

## Import Resources (dashboards, datasets, charts)

For ZIP-based import resources (like `superset_dashboard_import`), the pattern is more involved:

- Implement `resource.ResourceWithModifyPlan` to compute file hashes in `ModifyPlan`
- Store a `file_hashes` map in state (type `types.Map`, element type `types.StringType`)
- Use helpers from `internal/provider/import_helpers.go` for ZIP creation, hashing, and
  skip-pattern matching
- Implement retry logic for UUID lookup after import (Superset indexing can lag)
- See `dashboard_import_resource.go` as the canonical reference
