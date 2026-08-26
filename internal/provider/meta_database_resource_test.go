// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMetaDatabaseResource covers the full CRUD + ImportState + Update lifecycle
// of the superset_meta_database resource against a real Superset instance.
func TestAccMetaDatabaseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_meta_database" "test" {
  database_name = "tf-acc-meta-db"

  allowed_databases = ["db1", "db2"]

  expose_in_sqllab      = true
  allow_ctas            = false
  allow_cvas            = false
  allow_dml             = false
  allow_run_async       = true
  is_managed_externally = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_meta_database.test", "database_name", "tf-acc-meta-db"),
					resource.TestCheckResourceAttr("superset_meta_database.test", "sqlalchemy_uri", "superset://"),
					resource.TestCheckResourceAttr("superset_meta_database.test", "allowed_databases.#", "2"),
					resource.TestCheckResourceAttr("superset_meta_database.test", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttrSet("superset_meta_database.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_meta_database.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: change allowed_databases and flip expose_in_sqllab
			{
				Config: providerConfig() + `
resource "superset_meta_database" "test" {
  database_name = "tf-acc-meta-db"

  allowed_databases = ["db1", "db2", "db3"]

  expose_in_sqllab      = false
  allow_ctas            = false
  allow_cvas            = false
  allow_dml             = false
  allow_run_async       = true
  is_managed_externally = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_meta_database.test", "allowed_databases.#", "3"),
					resource.TestCheckResourceAttr("superset_meta_database.test", "expose_in_sqllab", "false"),
				),
			},
		},
	})
}

// TestAccMetaDatabaseResourceDefaults verifies that omitting optional boolean
// fields applies the documented defaults.
func TestAccMetaDatabaseResourceDefaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "superset_meta_database" "defaults" {
  database_name     = "tf-acc-meta-db-defaults"
  allowed_databases = ["minimal_db"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "database_name", "tf-acc-meta-db-defaults"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "sqlalchemy_uri", "superset://"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "allowed_databases.#", "1"),
					// Default values
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "expose_in_sqllab", "true"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "allow_ctas", "false"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "allow_cvas", "false"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "allow_dml", "false"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "allow_run_async", "true"),
					resource.TestCheckResourceAttr("superset_meta_database.defaults", "is_managed_externally", "false"),
					resource.TestCheckResourceAttrSet("superset_meta_database.defaults", "id"),
				),
			},
		},
	})
}
