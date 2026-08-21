# Testing

## Overview

All tests in `internal/provider/` are acceptance tests run with `TF_ACC=1`. They use `httpmock` to 
intercept HTTP calls — no real Superset instance is needed. Client unit tests in `internal/client/` 
also use `httpmock` directly.

## Running Tests

```bash
# Run all acceptance tests (primary target)
make testacc

# Run with filter
make testacc TESTARGS="-run TestAccRoleResource"

# Run a single package without the Makefile
TF_ACC=1 go test ./internal/provider/ -v -run TestAccCSSTemplate -timeout 120m

# Run client unit tests (no TF_ACC needed)
go test ./internal/client/ -v
```

## Shared Test Setup (`provider_test.go`)

Every test file imports two shared helpers:

```go
// providerConfig is the HCL block added before every resource/data-source config in test steps
const providerConfig = `
provider "superset" {
  host     = "http://superset-host"
  username = "fake-username"
  password = "fake-password"
}
`

// testAccProtoV6ProviderFactories wires the in-process provider for terraform-plugin-testing
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "superset": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
    // Add any pre-check logic here (env var validation, etc.)
}
```

The provider connects to `http://superset-host` with fake credentials. Every test must mock all HTTP calls that the provider will make against that host.

## Acceptance Test Pattern

```go
package provider_test

import (
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
    "github.com/jarcoal/httpmock"
)

func TestAccFooResource(t *testing.T) {
    // 1. Activate httpmock — intercepts all http.DefaultClient requests
    httpmock.Activate()
    defer httpmock.DeactivateAndReset()

    // 2. Mock authentication (always required — provider authenticates on configure)
    httpmock.RegisterResponder("POST", "http://superset-host/api/v1/security/login",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
            "access_token": "fake-token",
        }),
    )

    // 3. Mock CSRF token endpoint (required for all mutating calls)
    httpmock.RegisterResponder("GET", "http://superset-host/api/v1/security/csrf_token/",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
            "result": "fake-csrf-token",
        }),
    )

    // 4. Mock the actual API endpoints your resource calls
    httpmock.RegisterResponder("POST", "http://superset-host/api/v1/foo/",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
            "id": 42,
            "result": map[string]interface{}{
                "name": "test-foo",
            },
        }),
    )
    httpmock.RegisterResponder("GET", `=~^http://superset-host/api/v1/foo/\d+$`,
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
            "result": map[string]interface{}{
                "id":   42,
                "name": "test-foo",
            },
        }),
    )
    httpmock.RegisterResponder("DELETE", `=~^http://superset-host/api/v1/foo/\d+$`,
        httpmock.NewStringResponder(200, "{}"),
    )

    // 5. Run the test
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: providerConfig + `
resource "superset_foo" "test" {
  name = "test-foo"
}`,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("superset_foo.test", "name", "test-foo"),
                    resource.TestCheckResourceAttrSet("superset_foo.test", "id"),
                ),
            },
            // ImportState
            {
                ResourceName:      "superset_foo.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## httpmock Patterns

### Exact URL match
```go
httpmock.RegisterResponder("GET", "http://superset-host/api/v1/security/roles?q=(page_size:5000)",
    httpmock.NewJsonResponderOrPanic(200, responseBody),
)
```

### Regex URL match (for dynamic IDs)
```go
httpmock.RegisterResponder("GET", `=~^http://superset-host/api/v1/role/\d+$`,
    httpmock.NewJsonResponderOrPanic(200, responseBody),
)
```

### Ordered responders (for resources that call the same endpoint multiple times)
```go
// First call returns 42, second call returns 42 with updated data
httpmock.RegisterResponderWithQuery("GET", "http://superset-host/api/v1/foo/42", nil,
    httpmock.NewJsonResponderOrPanic(200, firstResponse),
)
```

Or use `httpmock.ResponderFromMultipleResponses` for sequences:
```go
httpmock.RegisterResponder("GET", `=~^http://superset-host/api/v1/foo/\d+$`,
    httpmock.NewJsonResponderOrPanic(200, responseBody), // reused for every call
)
```

### 404 response (resource deleted outside Terraform)
```go
httpmock.RegisterResponder("GET", "http://superset-host/api/v1/foo/42",
    httpmock.NewStringResponder(404, `{"message": "Not Found"}`),
)
```

## What to Mock

For any resource test, you need to mock every HTTP call the provider makes during the test. The common set is:

| When | Endpoint | Method |
|---|---|---|
| Always (provider configure) | `/api/v1/security/login` | POST |
| Before every mutating call | `/api/v1/security/csrf_token/` | GET |
| Create | Resource-specific POST endpoint | POST |
| Read (after create, before each step) | Resource-specific GET endpoint | GET |
| Update | Resource-specific PUT/POST endpoint | PUT/POST |
| Delete (destroy step) | Resource-specific DELETE endpoint | DELETE |
| ImportState | Resource-specific GET endpoint | GET |

## Client Unit Tests (`internal/client/`)

Client tests use httpmock directly without the Terraform testing framework:

```go
func TestCreateFoo(t *testing.T) {
    httpmock.Activate()
    defer httpmock.DeactivateAndReset()

    // Mock login
    httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/login",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{"access_token": "tok"}),
    )

    c, err := NewClient("http://test-host", "user", "pass", "db")
    assert.NoError(t, err)

    httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{"result": "csrf"}),
    )
    httpmock.RegisterResponder("POST", "http://test-host/api/v1/foo/",
        httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{"id": float64(1)}),
    )

    id, err := c.CreateFoo("my-foo")
    assert.NoError(t, err)
    assert.Equal(t, int64(1), id)
}
```

## Global Database Cache in Tests

The `GetAllDatabases` client method uses a global in-process cache with a 5-minute TTL. Tests that exercise database-related code must clear this cache:

```go
func TestAccDatabaseResource(t *testing.T) {
    client.ClearGlobalDatabaseCache()
    // ...
}
```

## Test File Naming

- Provider acceptance tests: `internal/provider/<name>_resource_test.go`
- Data source tests: `internal/provider/<name>_data_source_test.go`
- Client unit tests: `internal/client/<name>_test.go`
- All test files use `package provider_test` (external test package) for provider tests
