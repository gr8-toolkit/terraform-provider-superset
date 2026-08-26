// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDatasetsDataSource seeds one dataset then reads the full list via the
// superset_datasets data source.
func TestAccDatasetsDataSource(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "superset_database" "ds_db" {
  connection_name  = "tf-acc-datasets-ds-db"
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

resource "superset_dataset" "ds_seed" {
  table_name    = "ab_user"
  database_name = superset_database.ds_db.connection_name
  schema        = "public"
  depends_on    = [superset_database.ds_db]
}

data "superset_datasets" "test" {
  depends_on = [superset_dataset.ds_seed]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.superset_datasets.test", "datasets.#"),
				),
			},
		},
	})
}
