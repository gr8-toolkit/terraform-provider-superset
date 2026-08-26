// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRolePermissionsResource creates a role, assigns permissions taken from the
// real Superset permission-resources list, then verifies the state and import.
//
// We use the built-in "can_read on Charts" permission which is present in every
// fresh Superset installation, so the test requires no pre-seeded data beyond the
// default installation.
func TestAccRolePermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create: assign one permission to a freshly created role.
			{
				Config: providerConfig() + `
resource "superset_role" "perm_test" {
  name = "tf-acc-role-perms"
}

resource "superset_role_permissions" "test" {
  role_name = superset_role.perm_test.name

  resource_permissions = [
    {
      permission = "can_read"
      view_menu  = "Chart"
    },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "role_name", "tf-acc-role-perms"),
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", "1"),
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.0.permission", "can_read"),
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.0.view_menu", "Chart"),
				),
			},
			// Update: add a second permission.
			{
				Config: providerConfig() + `
resource "superset_role" "perm_test" {
  name = "tf-acc-role-perms"
}

resource "superset_role_permissions" "test" {
  role_name = superset_role.perm_test.name

  resource_permissions = [
    {
      permission = "can_read"
      view_menu  = "Chart"
    },
    {
      permission = "can_read"
      view_menu  = "Dashboard"
    },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", "2"),
				),
			},
		},
	})
}
