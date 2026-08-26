// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRoleDataSource creates a role and then looks it up via the data source.
func TestAccRoleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a role first so the data source is guaranteed to find it.
				Config: providerConfig() + `
resource "superset_role" "ds_test" {
  name = "tf-acc-role-ds"
}

data "superset_role" "test" {
  name       = superset_role.ds_test.name
  depends_on = [superset_role.ds_test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.superset_role.test", "name", "tf-acc-role-ds"),
					resource.TestCheckResourceAttrSet("data.superset_role.test", "id"),
				),
			},
		},
	})
}
