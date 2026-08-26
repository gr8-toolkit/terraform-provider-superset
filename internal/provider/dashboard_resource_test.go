// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDashboardResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_dashboard" "test" {
  dashboard_title = "tf-acc-dashboard"
  published       = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dashboard.test", "dashboard_title", "tf-acc-dashboard"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "published", "false"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "id"),
					resource.TestCheckResourceAttrSet("superset_dashboard.test", "uuid"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_dashboard.test",
				ImportState:       true,
				ImportStateVerify: true,
				// position_json and json_metadata may be normalised by Superset on create.
				ImportStateVerifyIgnore: []string{"position_json", "json_metadata"},
			},
			// Update: publish and add slug
			{
				Config: providerConfig() + `
resource "superset_dashboard" "test" {
  dashboard_title = "tf-acc-dashboard"
  slug            = "tf-acc-dashboard"
  published       = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_dashboard.test", "published", "true"),
					resource.TestCheckResourceAttr("superset_dashboard.test", "slug", "tf-acc-dashboard"),
				),
			},
		},
	})
}
