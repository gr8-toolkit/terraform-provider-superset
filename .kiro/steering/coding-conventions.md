# Coding Conventions

## Resource File Structure

Every resource lives in `internal/provider/<name>_resource.go`. Follow this exact layout:

```go
package provider

import (
    "context"
    "fmt"
    "strconv"

    "terraform-provider-superset/internal/client"

    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// 1. Compile-time interface assertions — always include these.
var (
    _ resource.Resource                = &fooResource{}
    _ resource.ResourceWithConfigure   = &fooResource{}
    _ resource.ResourceWithImportState = &fooResource{} // omit if import is not supported
)

// 2. Constructor — used in provider.go Resources() slice.
func NewFooResource() resource.Resource {
    return &fooResource{}
}

// 3. Resource struct — only holds the client.
type fooResource struct {
    client *client.Client
}

// 4. Model struct — one field per schema attribute, tfsdk tags must match schema keys exactly.
type fooResourceModel struct {
    ID   types.Int64  `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
    // sensitive fields still appear in the model — Sensitive: true is set in Schema
}

// 5. Metadata — TypeName = ProviderTypeName + "_foo"
func (r *fooResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_foo"
}

// 6. Schema — see Schema Conventions below
// 7. Configure — see Configure Pattern below
// 8. Create / Read / Update / Delete — see CRUD Patterns below
// 9. ImportState — see Import Pattern below
```

## Schema Conventions

```go
func (r *fooResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a foo in Superset.", // always provide a top-level description
        Attributes: map[string]schema.Attribute{
            // ID: always Computed, always UseStateForUnknown
            "id": schema.Int64Attribute{
                Description: "Numeric identifier of the foo.",
                Computed:    true,
                PlanModifiers: []planmodifier.Int64{
                    int64planmodifier.UseStateForUnknown(),
                },
            },
            // Required string
            "name": schema.StringAttribute{
                Description: "Name of the foo.",
                Required:    true,
            },
            // Optional+Computed with default
            "active": schema.BoolAttribute{
                Description: "Whether the foo is active.",
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(true),
            },
            // Sensitive field
            "password": schema.StringAttribute{
                Description: "Password for the foo.",
                Optional:    true,
                Sensitive:   true,
            },
            // Nested block
            "config": schema.SingleNestedAttribute{
                Optional: true,
                Attributes: map[string]schema.Attribute{
                    "key": schema.StringAttribute{Required: true},
                },
            },
        },
    }
}
```

Rules:

- Every attribute must have a `Description`.
- Computed IDs always get `UseStateForUnknown()` — prevents unnecessary diffs after create.
- Passwords, URIs, encrypted extras, and secrets always get `Sensitive: true`.
- Use `booldefault.StaticBool(...)` for boolean fields with a meaningful default.
- Prefer `schema.Int64Attribute` for numeric IDs (Superset IDs are integers).
- `RequiresReplace()` plan modifier when changing a field must recreate the resource.

## Configure Pattern

Every resource and data source implements `Configure` the same way:

```go
func (r *fooResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*client.Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
        )
        return
    }
    r.client = c
}
```

## CRUD Patterns

### Create

```go
func (r *fooResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    tflog.Debug(ctx, "Starting Create method")

    var plan fooResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    id, err := r.client.CreateFoo(plan.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Unable to Create Superset Foo", fmt.Sprintf("CreateFoo failed: %s", err.Error()))
        return
    }

    plan.ID = types.Int64Value(id)

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)

    tflog.Debug(ctx, fmt.Sprintf("Created foo: ID=%d, Name=%s", plan.ID.ValueInt64(), plan.Name.ValueString()))
}
```

### Read

```go
func (r *fooResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    tflog.Debug(ctx, "Starting Read method")

    var state fooResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    foo, err := r.client.GetFoo(state.ID.ValueInt64())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error reading foo", fmt.Sprintf("Could not read foo ID %d: %s", state.ID.ValueInt64(), err.Error())
        )
        return
    }

    state.Name = types.StringValue(foo.Name)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Delete

```go
func (r *fooResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    tflog.Debug(ctx, "Starting Delete method")

    var state fooResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    err := r.client.DeleteFoo(state.ID.ValueInt64())
    if err != nil {
        // Treat 404 as already deleted — idempotent
        if err.Error() == "failed to delete foo, status code: 404" {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Unable to Delete Superset Foo", fmt.Sprintf("DeleteFoo failed: %s", err.Error()))
        return
    }

    resp.State.RemoveResource(ctx)
    tflog.Debug(ctx, fmt.Sprintf("Deleted foo: ID=%d", state.ID.ValueInt64()))
}
```

### ImportState

```go
func (r *fooResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    tflog.Debug(ctx, "Starting ImportState method", map[string]interface{}{
        "import_id": req.ID,
    })

    id, err := strconv.ParseInt(req.ID, 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Invalid Import ID",
            fmt.Sprintf("The provided import ID '%s' is not a valid int64: %s", req.ID, err.Error()),
        )
        return
    }

    resp.State.SetAttribute(ctx, path.Root("id"), id)
}
```

## Data Source Pattern

Data sources mirror resources but only have `Read`. All schema attributes are `Computed: true`.

```go
// File: internal/provider/foos_data_source.go

var _ datasource.DataSource              = &foosDataSource{}
var _ datasource.DataSourceWithConfigure = &foosDataSource{}

func NewFoosDataSource() datasource.DataSource { return &foosDataSource{} }

type foosDataSource struct { client *client.Client }

type foosDataSourceModel struct {
    Foos []fooModel `tfsdk:"foos"`
}

type fooModel struct {
    ID   types.Int64  `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
}

func (d *foosDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_foos"
}
```

Configure method is identical to resources. Read follows the same get-state → call client → set-state pattern.

## API Client Conventions (`internal/client/superset.go`)

### Authentication and CSRF

The client authenticates on construction (`NewClient` → `authenticate()`). The JWT token is stored as `c.Token`.

All mutating calls (POST/PUT/DELETE on most endpoints) require a CSRF token.
Fetch it first, then include it as a header and cookie:

```go
func (c *Client) CreateFoo(name string) (int64, error) {
    csrfToken, cookies, err := c.GetCSRFToken()
    if err != nil {
        return 0, fmt.Errorf("failed to get CSRF token: %w", err)
    }

    payload := map[string]interface{}{"name": name}
    headers := map[string]string{
        "X-CSRFToken": csrfToken,
        "Referer":     c.Host,
    }

    resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/foo/", payload, headers, cookies)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return 0, fmt.Errorf("failed to create foo, status code: %d, response: %s", resp.StatusCode, string(body))
    }

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }

    id, ok := result["id"].(float64) // JSON numbers unmarshal as float64
    if !ok {
        return 0, fmt.Errorf("invalid id in response")
    }
    return int64(id), nil
}
```

Read-only GET requests use `DoRequest` (no CSRF needed):

```go
func (c *Client) GetFoo(id int64) (*Foo, error) {
    resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/foo/%d", id), nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("foo %d not found", id)
    }
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to get foo, status code: %d, response: %s", resp.StatusCode, string(body))
    }
    // decode...
}
```

### Error Messages

Error strings from client methods follow this format:

```text
"failed to <verb> <entity>, status code: %d, response: %s"
"failed to <verb> <entity>, status code: 404"
```

Resources that handle 404 as "already deleted" check for the exact string:

```go
if err.Error() == "failed to delete foo, status code: 404" {
    resp.State.RemoveResource(ctx)
    return
}
```

### JSON Type Assertions

Superset API responses are decoded into `map[string]interface{}`.
JSON numbers always unmarshal as `float64` in Go — convert explicitly:

```go
id := int64(result["id"].(float64))
name := result["name"].(string)
```

Use comma-ok form when the field is optional or might be absent:

```go
if val, ok := resultData["cache_timeout"].(float64); ok {
    state.CacheTimeout = types.Int64Value(int64(val))
}
```

### Logging

Always use `tflog` — never `fmt.Printf` or `log.Printf`:

```go
tflog.Debug(ctx, "Starting Create method")
tflog.Debug(ctx, fmt.Sprintf("Created foo: ID=%d", id))
tflog.Warn(ctx, "Foo not found, removing from state", map[string]interface{}{"id": state.ID.ValueInt64()})
```

Existing `fmt.Printf("DEBUG ...")` calls in `superset.go` are a known issue —
replace them with `tflog.Debug` when editing those methods.

## Handling Optional Fields in State

Guard every optional field before reading its value:

```go
if !plan.SomeField.IsNull() && !plan.SomeField.IsUnknown() {
    value = plan.SomeField.ValueString()
}
```

When a computed field might not be returned by the API, fall back to zero/empty rather than leaving it Unknown:

```go
if val, ok := resultData["cache_timeout"].(float64); ok {
    state.CacheTimeout = types.Int64Value(int64(val))
} else if state.CacheTimeout.IsUnknown() {
    state.CacheTimeout = types.Int64Value(0)
}
```

## Provider Registration

When adding a new resource or data source, register it in `internal/provider/provider.go`:

```go
// Resources
func (p *supersetProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ...existing...
        NewFooResource, // add here
    }
}

// Data sources
func (p *supersetProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ...existing...
        NewFoosDataSource, // add here
    }
}
```
