// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccChartResource(t *testing.T) {
	client.ClearGlobalDatabaseCache()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_database" "chart_db" {
  connection_name  = "tf-acc-chart-db"
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

resource "superset_dataset" "chart_ds" {
  table_name    = "ab_user"
  database_name = superset_database.chart_db.connection_name
  schema        = "public"
  depends_on    = [superset_database.chart_db]
}

resource "superset_chart" "test" {
  slice_name    = "tf-acc-chart"
  viz_type      = "table"
  datasource_id = superset_dataset.chart_ds.id
  params        = jsonencode({ metrics = ["count"] })
  description   = "Acceptance test chart"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_chart.test", "slice_name", "tf-acc-chart"),
					resource.TestCheckResourceAttr("superset_chart.test", "viz_type", "table"),
					resource.TestCheckResourceAttr("superset_chart.test", "description", "Acceptance test chart"),
					resource.TestCheckResourceAttr("superset_chart.test", "datasource_type", "table"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "id"),
					resource.TestCheckResourceAttrSet("superset_chart.test", "uuid"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_chart.test",
				ImportState:       true,
				ImportStateVerify: true,
				// query_context is auto-generated server-side on create and may differ.
				// cache_timeout: API returns null which maps to 0, but the plan had no
				// value set so the import verify sees a mismatch.
				ImportStateVerifyIgnore: []string{"query_context", "cache_timeout"},
			},
			// Update: rename and change description
			{
				Config: providerConfig() + `
resource "superset_database" "chart_db" {
  connection_name  = "tf-acc-chart-db"
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

resource "superset_dataset" "chart_ds" {
  table_name    = "ab_user"
  database_name = superset_database.chart_db.connection_name
  schema        = "public"
  depends_on    = [superset_database.chart_db]
}

resource "superset_chart" "test" {
  slice_name    = "tf-acc-chart-updated"
  viz_type      = "table"
  datasource_id = superset_dataset.chart_ds.id
  params        = jsonencode({ metrics = ["count"] })
  description   = "Updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_chart.test", "slice_name", "tf-acc-chart-updated"),
					resource.TestCheckResourceAttr("superset_chart.test", "description", "Updated description"),
				),
			},
		},
	})
}
