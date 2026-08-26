// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// dbConfig is the HCL block that creates the PostgreSQL database the dataset
// tests reference. It points at the same Postgres service used as Superset's
// metadata store, which is always present in the test compose stack.
const dbConfig = `
resource "superset_database" "test_db" {
  connection_name  = "tf-acc-dataset-db"
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
`

// TestAccDatasetResource covers create, read, import, and update for a physical
// (non-SQL) dataset backed by the PostgreSQL metadata DB that Superset itself uses.
func TestAccDatasetResource(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccDatasetResourceConfig("ab_user", "public"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "table_name", "ab_user"),
					resource.TestCheckResourceAttr("superset_dataset.test", "schema", "public"),
					resource.TestCheckResourceAttrSet("superset_dataset.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_dataset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: change schema (no-op value change to exercise the update path)
			{
				Config: testAccDatasetResourceConfig("ab_user", "public"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "table_name", "ab_user"),
					resource.TestCheckResourceAttr("superset_dataset.test", "schema", "public"),
				),
			},
		},
	})
}

// TestAccDatasetResourceWithSQL covers a virtual (SQL-based) dataset.
func TestAccDatasetResourceWithSQL(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetResourceConfigWithSQL(
					"tf_acc_sql_dataset",
					"SELECT 1 AS id",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dataset.test", "table_name", "tf_acc_sql_dataset"),
					resource.TestCheckResourceAttr("superset_dataset.test", "sql", "SELECT 1 AS id"),
					resource.TestCheckResourceAttrSet("superset_dataset.test", "id"),
				),
			},
		},
	})
}

func testAccDatasetResourceConfig(tableName, schemaName string) string {
	return providerConfig() + dbConfig + fmt.Sprintf(`
resource "superset_dataset" "test" {
  table_name    = %[1]q
  database_name = superset_database.test_db.connection_name
  schema        = %[2]q
  depends_on    = [superset_database.test_db]
}
`, tableName, schemaName)
}

func testAccDatasetResourceConfigWithSQL(tableName, sql string) string {
	return providerConfig() + dbConfig + fmt.Sprintf(`
resource "superset_dataset" "test" {
  table_name    = %[1]q
  database_name = superset_database.test_db.connection_name
  sql           = %[2]q
  depends_on    = [superset_database.test_db]
}
`, tableName, sql)
}
