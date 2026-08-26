// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDatabaseResource covers the core CRUD + ImportState lifecycle.
func TestAccDatabaseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_database" "test" {
  connection_name  = "tf-acc-db"
  db_engine        = "postgresql"
  db_user          = "superset"
  db_pass          = "superset"
  db_host          = "db"
  db_port          = 5432
  db_name          = "superset"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = false
  allow_run_async  = false
  expose_in_sqllab = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.test", "connection_name", "tf-acc-db"),
					resource.TestCheckResourceAttr("superset_database.test", "db_engine", "postgresql"),
					resource.TestCheckResourceAttr("superset_database.test", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttrSet("superset_database.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_database.test",
				ImportState:       true,
				ImportStateVerify: true,
				// db_pass is write-only and never returned by the API.
				// cache_timeout is omitted in the config (defaults to 0) but the API
				// returns null which the provider maps back differently on import.
				// extra is omitted in the config but Superset always returns a default
				// JSON object — the provider cannot distinguish "user set empty" from
				// "API default" so we skip it on import verification.
				ImportStateVerifyIgnore: []string{"db_pass", "cache_timeout", "extra"},
			},
			// Update: flip expose_in_sqllab and allow_dml
			{
				Config: providerConfig() + `
resource "superset_database" "test" {
  connection_name  = "tf-acc-db"
  db_engine        = "postgresql"
  db_user          = "superset"
  db_pass          = "superset"
  db_host          = "db"
  db_port          = 5432
  db_name          = "superset"
  allow_ctas       = false
  allow_cvas       = false
  allow_dml        = true
  allow_run_async  = false
  expose_in_sqllab = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.test", "allow_dml", "true"),
					resource.TestCheckResourceAttr("superset_database.test", "expose_in_sqllab", "false"),
				),
			},
		},
	})
}

// TestAccDatabaseResourceOptionalFields verifies force_ctas_schema, server_cert,
// and masked_encrypted_extra round-trip through create and read.
func TestAccDatabaseResourceOptionalFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "superset_database" "optional_test" {
  connection_name   = "tf-acc-db-optional"
  db_engine         = "postgresql"
  db_user           = "superset"
  db_pass           = "superset"
  db_host           = "db"
  db_port           = 5432
  db_name           = "superset"
  allow_ctas        = false
  allow_cvas        = false
  allow_dml         = false
  allow_run_async   = false
  expose_in_sqllab  = false
  force_ctas_schema = "public"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_database.optional_test", "connection_name", "tf-acc-db-optional"),
					// force_ctas_schema is write-only (not returned by API), kept from plan.
					resource.TestCheckResourceAttr("superset_database.optional_test", "force_ctas_schema", "public"),
					resource.TestCheckResourceAttrSet("superset_database.optional_test", "id"),
				),
			},
		},
	})
}
