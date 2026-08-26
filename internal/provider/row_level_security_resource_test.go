// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRowLevelSecurityResource covers the full CRUD + ImportState lifecycle.
// An RLS rule references a dataset and a role by ID, so both are created here.
func TestAccRowLevelSecurityResource(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
data "superset_role" "gamma" {
  name = "Gamma"
}

resource "superset_database" "rls_db" {
  connection_name  = "tf-acc-rls-db"
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
  expose_in_sqllab = false
}

resource "superset_dataset" "rls_seed" {
  table_name    = "ab_user"
  database_name = superset_database.rls_db.connection_name
  schema        = "public"
  depends_on    = [superset_database.rls_db]
}

resource "superset_row_level_security" "test" {
  name        = "tf-acc-rls"
  tables      = [superset_dataset.rls_seed.id]
  clause      = "1=1"
  role_ids    = [data.superset_role.gamma.id]
  group_key   = "tf-acc"
  filter_type = "Regular"
  description = "Acceptance test RLS rule"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_row_level_security.test", "name", "tf-acc-rls"),
					resource.TestCheckResourceAttr("superset_row_level_security.test", "clause", "1=1"),
					resource.TestCheckResourceAttr("superset_row_level_security.test", "group_key", "tf-acc"),
					resource.TestCheckResourceAttr("superset_row_level_security.test", "filter_type", "Regular"),
					resource.TestCheckResourceAttr("superset_row_level_security.test", "description", "Acceptance test RLS rule"),
					resource.TestCheckResourceAttrSet("superset_row_level_security.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_row_level_security.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
