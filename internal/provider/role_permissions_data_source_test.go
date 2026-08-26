// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRolePermissionsDataSource creates a role, assigns a permission, then reads
// that back via the data source.
func TestAccRolePermissionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "superset_role" "ds_perms" {
  name = "tf-acc-role-perms-ds"
}

resource "superset_role_permissions" "ds_perms" {
  role_name = superset_role.ds_perms.name

  resource_permissions = [
    {
      permission = "can_read"
      view_menu  = "Chart"
    },
  ]
}

data "superset_role_permissions" "test" {
  role_name  = superset_role.ds_perms.name
  depends_on = [superset_role_permissions.ds_perms]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "role_name", "tf-acc-role-perms-ds"),
					resource.TestCheckResourceAttrSet("data.superset_role_permissions.test", "permissions.#"),
				),
			},
		},
	})
}
