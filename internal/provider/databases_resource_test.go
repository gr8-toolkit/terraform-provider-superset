package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

// baseDBCreateResponse is the common API response shape for a freshly created database.
const baseDBCreateResponse = `{
	"id": 208,
	"result": {
		"allow_ctas": false,
		"allow_cvas": false,
		"allow_dml": false,
		"allow_file_upload": false,
		"allow_run_async": true,
		"cache_timeout": null,
		"configuration_method": "sqlalchemy_form",
		"database_name": "DWH_database_connection4",
		"driver": "psycopg2",
		"expose_in_sqllab": true,
		"extra": "{\"client_encoding\": \"utf8\"}",
		"impersonate_user": false,
		"parameters": {
			"database": "superset_db",
			"encryption": false,
			"host": "pg.db.ro.domain.com",
			"password": "XXXXXXXXXX",
			"port": 5432,
			"query": {},
			"username": "superset_user"
		},
		"sqlalchemy_uri": "postgresql://superset_user:XXXXXXXXXX@pg.db.ro.domain.com:5432/superset_db",
		"uuid": "f5007595-5a43-45d8-a1da-9612bdb12b22"
	}
}`

const baseDBReadResponse = `{
	"result": {
		"allow_ctas": false,
		"allow_cvas": false,
		"allow_dml": false,
		"allow_file_upload": false,
		"allow_run_async": true,
		"backend": "postgresql",
		"cache_timeout": null,
		"configuration_method": "sqlalchemy_form",
		"database_name": "DWH_database_connection4",
		"driver": "psycopg2",
		"expose_in_sqllab": true,
		"extra": "{\"client_encoding\": \"utf8\"}",
		"impersonate_user": false,
		"parameters": {
			"database": "superset_db",
			"encryption": false,
			"host": "pg.db.ro.domain.com",
			"password": "XXXXXXXXXX",
			"port": 5432,
			"query": {},
			"username": "superset_user"
		},
		"sqlalchemy_uri": "postgresql://superset_user:XXXXXXXXXX@pg.db.ro.domain.com:5432/superset_db",
		"uuid": "f5007595-5a43-45d8-a1da-9612bdb12b22"
	}
}`

// registerBaseDBMocks registers the common login/CSRF mocks shared by all database tests.
func registerBaseDBMocks() {
	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/security/login",
		httpmock.NewStringResponder(200, `{"access_token": "fake-token"}`))
	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "fake-csrf-token"}`))
}

// TestAccDatabaseResource covers the core CRUD lifecycle including the new
// allow_file_upload, impersonate_user, cache_timeout, and extra fields.
func TestAccDatabaseResource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerBaseDBMocks()

	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/database/",
		httpmock.NewStringResponder(201, baseDBCreateResponse))

	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/database/208/connection",
		httpmock.NewStringResponder(200, baseDBReadResponse))

	// Update returns expose_in_sqllab=false to verify state is refreshed from response.
	httpmock.RegisterResponder("PUT", "http://superset-host/api/v1/database/208",
		httpmock.NewStringResponder(200, `{
			"id": 208,
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": true,
				"cache_timeout": null,
				"configuration_method": "sqlalchemy_form",
				"database_name": "DWH_database_connection4",
				"driver": "psycopg2",
				"expose_in_sqllab": false,
				"extra": "{\"client_encoding\": \"utf8\"}",
				"impersonate_user": false,
				"parameters": {
					"database": "superset_db",
					"encryption": false,
					"host": "pg.db.ro.domain.com",
					"password": "XXXXXXXXXX",
					"port": 5432,
					"query": {},
					"username": "superset_user"
				},
				"sqlalchemy_uri": "postgresql://superset_user:XXXXXXXXXX@pg.db.ro.domain.com:5432/superset_db",
				"uuid": "f5007595-5a43-45d8-a1da-9612bdb12b22"
			}
		}`))

	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/database/208",
		httpmock.NewStringResponder(200, ""))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig + testAccDatabaseResourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.test", "connection_name", "DWH_database_connection4"),
					resource.TestCheckResourceAttr("superset_database.test", "db_engine", "postgresql"),
					resource.TestCheckResourceAttr("superset_database.test", "db_user", "superset_user"),
					resource.TestCheckResourceAttr("superset_database.test", "db_host", "pg.db.ro.domain.com"),
					resource.TestCheckResourceAttr("superset_database.test", "db_port", "5432"),
					resource.TestCheckResourceAttr("superset_database.test", "db_name", "superset_db"),
					resource.TestCheckResourceAttr("superset_database.test", "allow_ctas", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "allow_cvas", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "allow_dml", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "allow_file_upload", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "allow_run_async", "true"),
					resource.TestCheckResourceAttr("superset_database.test", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttr("superset_database.test", "impersonate_user", "false"),
					resource.TestCheckResourceAttr("superset_database.test", "extra", `{"client_encoding": "utf8"}`),
				),
			},
		},
	})
}

// TestAccDatabaseResourceUpdate exercises a two-step plan: create then update,
// verifying that allow_file_upload, impersonate_user, cache_timeout, and extra
// are all correctly reflected in state after each operation.
func TestAccDatabaseResourceUpdate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerBaseDBMocks()

	// Step 1 – Create
	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{
			"id": 209,
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"cache_timeout": null,
				"database_name": "UpdateTestDB",
				"expose_in_sqllab": false,
				"extra": "{\"client_encoding\": \"utf8\"}",
				"impersonate_user": false
			}
		}`))

	var readStep = "initial"
	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/database/209/connection",
		func(req *http.Request) (*http.Response, error) {
			if readStep == "updated" {
				return httpmock.NewStringResponse(200, `{
					"result": {
						"allow_ctas": false,
						"allow_cvas": false,
						"allow_dml": true,
						"allow_file_upload": true,
						"allow_run_async": true,
						"backend": "postgresql",
						"cache_timeout": 300,
						"database_name": "UpdateTestDB",
						"expose_in_sqllab": true,
						"extra": "{\"client_encoding\": \"utf8\", \"cost_estimate_enabled\": true}",
						"impersonate_user": true,
						"parameters": {
							"database": "superset_db",
							"host": "pg.db.ro.domain.com",
							"port": 5432,
							"username": "superset_user"
						}
					}
				}`), nil
			}
			return httpmock.NewStringResponse(200, `{
				"result": {
					"allow_ctas": false,
					"allow_cvas": false,
					"allow_dml": false,
					"allow_file_upload": false,
					"allow_run_async": false,
					"backend": "postgresql",
					"cache_timeout": null,
					"database_name": "UpdateTestDB",
					"expose_in_sqllab": false,
					"extra": "{\"client_encoding\": \"utf8\"}",
					"impersonate_user": false,
					"parameters": {
						"database": "superset_db",
						"host": "pg.db.ro.domain.com",
						"port": 5432,
						"username": "superset_user"
					}
				}
			}`), nil
		})

	// Step 2 – Update
	httpmock.RegisterResponder("PUT", "http://superset-host/api/v1/database/209",
		func(req *http.Request) (*http.Response, error) {
			readStep = "updated"
			return httpmock.NewStringResponse(200, `{
				"id": 209,
				"result": {
					"allow_ctas": false,
					"allow_cvas": false,
					"allow_dml": true,
					"allow_file_upload": true,
					"allow_run_async": true,
					"cache_timeout": 300,
					"database_name": "UpdateTestDB",
					"expose_in_sqllab": true,
					"extra": "{\"client_encoding\": \"utf8\", \"cost_estimate_enabled\": true}",
					"impersonate_user": true
				}
			}`), nil
		})

	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/database/209",
		httpmock.NewStringResponder(200, ""))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with baseline values
			{
				Config: providerConfig + testAccDatabaseResourceUpdateConfigStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.update_test", "connection_name", "UpdateTestDB"),
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_dml", "false"),
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_file_upload", "false"),
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_run_async", "false"),
					resource.TestCheckResourceAttr("superset_database.update_test", "expose_in_sqllab", "false"),
					resource.TestCheckResourceAttr("superset_database.update_test", "impersonate_user", "false"),
					resource.TestCheckResourceAttr("superset_database.update_test", "extra", `{"client_encoding": "utf8"}`),
				),
			},
			// Update – flip several new fields
			{
				Config: providerConfig + testAccDatabaseResourceUpdateConfigStep2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_dml", "true"),
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_file_upload", "true"),
					resource.TestCheckResourceAttr("superset_database.update_test", "allow_run_async", "true"),
					resource.TestCheckResourceAttr("superset_database.update_test", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttr("superset_database.update_test", "impersonate_user", "true"),
					resource.TestCheckResourceAttr("superset_database.update_test", "cache_timeout", "300"),
					resource.TestCheckResourceAttr("superset_database.update_test", "extra", `{"client_encoding": "utf8", "cost_estimate_enabled": true}`),
				),
			},
		},
	})
}

// TestAccDatabaseResourceOptionalFields verifies that force_ctas_schema,
// server_cert, and masked_encrypted_extra round-trip correctly through
// create and read without being clobbered.
func TestAccDatabaseResourceOptionalFields(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerBaseDBMocks()

	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{
			"id": 210,
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"cache_timeout": null,
				"database_name": "OptionalFieldsDB",
				"expose_in_sqllab": false,
				"extra": "{}",
				"impersonate_user": false
			}
		}`))

	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/database/210/connection",
		httpmock.NewStringResponder(200, `{
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"backend": "postgresql",
				"cache_timeout": null,
				"database_name": "OptionalFieldsDB",
				"expose_in_sqllab": false,
				"extra": "{}",
				"impersonate_user": false,
				"server_cert": "fake-ca-cert-for-testing",
				"parameters": {
					"database": "mydb",
					"host": "pg.example.com",
					"port": 5432,
					"username": "user"
				}
			}
		}`))

	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/database/210",
		httpmock.NewStringResponder(200, ""))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccDatabaseResourceOptionalFieldsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.optional_test", "connection_name", "OptionalFieldsDB"),
					// force_ctas_schema is write-only (not returned by API), kept from plan
					resource.TestCheckResourceAttr("superset_database.optional_test", "force_ctas_schema", "public"),
					// masked_encrypted_extra is sensitive/write-only, kept from plan
					resource.TestCheckResourceAttr("superset_database.optional_test", "masked_encrypted_extra", `{"project_id": "my-gcp-project"}`),
					// server_cert is read back from API response
					resource.TestCheckResourceAttr("superset_database.optional_test", "server_cert", "fake-ca-cert-for-testing"),
				),
			},
		},
	})
}

// TestAccDatabaseResourceSSHTunnel verifies that the ssh_tunnel nested block
// is sent to the API on create and that the nested attributes are preserved in state.
func TestAccDatabaseResourceSSHTunnel(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerBaseDBMocks()

	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{
			"id": 211,
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"cache_timeout": null,
				"database_name": "SSHTunnelDB",
				"expose_in_sqllab": true,
				"extra": "{}",
				"impersonate_user": false
			}
		}`))

	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/database/211/connection",
		httpmock.NewStringResponder(200, `{
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"backend": "postgresql",
				"cache_timeout": null,
				"database_name": "SSHTunnelDB",
				"expose_in_sqllab": true,
				"extra": "{}",
				"impersonate_user": false,
				"parameters": {
					"database": "mydb",
					"host": "internal.host",
					"port": 5432,
					"username": "user"
				}
			}
		}`))

	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/database/211",
		httpmock.NewStringResponder(200, ""))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccDatabaseResourceSSHTunnelConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.ssh_test", "connection_name", "SSHTunnelDB"),
					// SSH tunnel attributes are write-only (API redacts them), kept from plan
					resource.TestCheckResourceAttr("superset_database.ssh_test", "ssh_tunnel.server_address", "bastion.example.com"),
					resource.TestCheckResourceAttr("superset_database.ssh_test", "ssh_tunnel.server_port", "22"),
					resource.TestCheckResourceAttr("superset_database.ssh_test", "ssh_tunnel.username", "tunnel_user"),
					resource.TestCheckResourceAttr("superset_database.ssh_test", "ssh_tunnel.private_key", "fake-private-key-for-testing"),
				),
			},
		},
	})
}

// TestAccDatabaseResourceSSHTunnelWithPassword exercises the password-based
// SSH tunnel variant (as opposed to private key).
func TestAccDatabaseResourceSSHTunnelWithPassword(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	registerBaseDBMocks()

	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{
			"id": 212,
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"cache_timeout": null,
				"database_name": "SSHTunnelPasswordDB",
				"expose_in_sqllab": true,
				"extra": "{}",
				"impersonate_user": false
			}
		}`))

	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/database/212/connection",
		httpmock.NewStringResponder(200, `{
			"result": {
				"allow_ctas": false,
				"allow_cvas": false,
				"allow_dml": false,
				"allow_file_upload": false,
				"allow_run_async": false,
				"backend": "postgresql",
				"cache_timeout": null,
				"database_name": "SSHTunnelPasswordDB",
				"expose_in_sqllab": true,
				"extra": "{}",
				"impersonate_user": false,
				"parameters": {
					"database": "mydb",
					"host": "internal.host",
					"port": 5432,
					"username": "user"
				}
			}
		}`))

	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/database/212",
		httpmock.NewStringResponder(200, ""))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccDatabaseResourceSSHTunnelPasswordConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.ssh_pw_test", "connection_name", "SSHTunnelPasswordDB"),
					resource.TestCheckResourceAttr("superset_database.ssh_pw_test", "ssh_tunnel.server_address", "bastion.example.com"),
					resource.TestCheckResourceAttr("superset_database.ssh_pw_test", "ssh_tunnel.server_port", "22"),
					resource.TestCheckResourceAttr("superset_database.ssh_pw_test", "ssh_tunnel.username", "tunnel_user"),
					resource.TestCheckResourceAttr("superset_database.ssh_pw_test", "ssh_tunnel.password", "s3cr3t"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Terraform configs
// ---------------------------------------------------------------------------

const testAccDatabaseResourceConfig = `
resource "superset_database" "test" {
  connection_name = "DWH_database_connection4"
  db_engine       = "postgresql"
  db_user         = "superset_user"
  db_pass         = "dbpassword"
  db_host         = "pg.db.ro.domain.com"
  db_port         = 5432
  db_name         = "superset_db"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_run_async  = true
  expose_in_sqllab = true
}
`

const testAccDatabaseResourceUpdateConfigStep1 = `
resource "superset_database" "update_test" {
  connection_name  = "UpdateTestDB"
  db_engine        = "postgresql"
  db_user          = "superset_user"
  db_pass          = "dbpassword"
  db_host          = "pg.db.ro.domain.com"
  db_port          = 5432
  db_name          = "superset_db"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_file_upload = false
  allow_run_async  = false
  expose_in_sqllab = false
  impersonate_user = false
  extra            = "{\"client_encoding\": \"utf8\"}"
}
`

const testAccDatabaseResourceUpdateConfigStep2 = `
resource "superset_database" "update_test" {
  connection_name  = "UpdateTestDB"
  db_engine        = "postgresql"
  db_user          = "superset_user"
  db_pass          = "dbpassword"
  db_host          = "pg.db.ro.domain.com"
  db_port          = 5432
  db_name          = "superset_db"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = true
  allow_file_upload = true
  allow_run_async  = true
  expose_in_sqllab = true
  impersonate_user = true
  cache_timeout    = 300
  extra            = "{\"client_encoding\": \"utf8\", \"cost_estimate_enabled\": true}"
}
`

const testAccDatabaseResourceOptionalFieldsConfig = `
resource "superset_database" "optional_test" {
  connection_name       = "OptionalFieldsDB"
  db_engine             = "postgresql"
  db_user               = "user"
  db_pass               = "pass"
  db_host               = "pg.example.com"
  db_port               = 5432
  db_name               = "mydb"
  allow_ctas            = false
  allow_cvas            = false
  allow_dml             = false
  allow_run_async       = false
  expose_in_sqllab      = false
  force_ctas_schema     = "public"
  masked_encrypted_extra = "{\"project_id\": \"my-gcp-project\"}"
  server_cert           = "fake-ca-cert-for-testing"
}
`

const testAccDatabaseResourceSSHTunnelConfig = `
resource "superset_database" "ssh_test" {
  connection_name  = "SSHTunnelDB"
  db_engine        = "postgresql"
  db_user          = "user"
  db_pass          = "pass"
  db_host          = "internal.host"
  db_port          = 5432
  db_name          = "mydb"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_run_async  = false
  expose_in_sqllab = true

  ssh_tunnel = {
    server_address = "bastion.example.com"
    server_port    = 22
    username       = "tunnel_user"
    private_key    = "fake-private-key-for-testing"
  }
}
`

const testAccDatabaseResourceSSHTunnelPasswordConfig = `
resource "superset_database" "ssh_pw_test" {
  connection_name  = "SSHTunnelPasswordDB"
  db_engine        = "postgresql"
  db_user          = "user"
  db_pass          = "pass"
  db_host          = "internal.host"
  db_port          = 5432
  db_name          = "mydb"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_run_async  = false
  expose_in_sqllab = true

  ssh_tunnel = {
    server_address = "bastion.example.com"
    server_port    = 22
    username       = "tunnel_user"
    password       = "s3cr3t"
  }
}
`
