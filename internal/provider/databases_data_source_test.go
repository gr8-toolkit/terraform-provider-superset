// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDatabasesDataSource verifies the superset_databases data source returns
// the databases registered in the running Superset instance.
func TestAccDatabasesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Seed one database so the list is non-empty even on a pristine instance.
				Config: providerConfig() + `
resource "superset_database" "ds_seed" {
  connection_name  = "tf-acc-db-ds-seed"
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

data "superset_databases" "test" {
  depends_on = [superset_database.ds_seed]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.superset_databases.test", "databases.#"),
				),
			},
		},
	})
}
