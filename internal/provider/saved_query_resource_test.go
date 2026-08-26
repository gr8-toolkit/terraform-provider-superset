// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"terraform-provider-superset/internal/client"
)

func TestAccSavedQueryResource(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_database" "sq_db" {
  connection_name  = "tf-acc-sq-db"
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

resource "superset_saved_query" "test" {
  database_id = superset_database.sq_db.id
  label       = "tf-acc-saved-query"
  sql         = "SELECT 1 AS id"
  schema      = "public"
  description = "Acceptance test saved query"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_saved_query.test", "label", "tf-acc-saved-query"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "sql", "SELECT 1 AS id"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "schema", "public"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "description", "Acceptance test saved query"),
					resource.TestCheckResourceAttrSet("superset_saved_query.test", "id"),
					resource.TestCheckResourceAttrSet("superset_saved_query.test", "database_name"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_saved_query.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: change SQL and description
			{
				Config: providerConfig() + `
resource "superset_database" "sq_db" {
  connection_name  = "tf-acc-sq-db"
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

resource "superset_saved_query" "test" {
  database_id = superset_database.sq_db.id
  label       = "tf-acc-saved-query"
  sql         = "SELECT 2 AS id"
  schema      = "public"
  description = "Updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_saved_query.test", "sql", "SELECT 2 AS id"),
					resource.TestCheckResourceAttr("superset_saved_query.test", "description", "Updated description"),
				),
			},
		},
	})
}
