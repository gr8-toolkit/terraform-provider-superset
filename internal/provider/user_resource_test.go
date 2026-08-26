// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserResource covers the full CRUD + ImportState + Update lifecycle.
// It uses the built-in "Gamma" role (id is stable across fresh Superset installs
// because Superset seeds roles in a fixed order: Admin=1, Public=2, Alpha=3, Gamma=4).
func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
# Look up the Gamma role so we get its real ID.
data "superset_role" "gamma" {
  name = "Gamma"
}

resource "superset_user" "test" {
  username   = "tf-acc-user"
  first_name = "TFAcc"
  last_name  = "User"
  email      = "tf-acc-user@example.com"
  password   = "Acc3ptanceT3st!"
  active     = true
  roles      = [data.superset_role.gamma.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_user.test", "username", "tf-acc-user"),
					resource.TestCheckResourceAttr("superset_user.test", "first_name", "TFAcc"),
					resource.TestCheckResourceAttr("superset_user.test", "last_name", "User"),
					resource.TestCheckResourceAttr("superset_user.test", "email", "tf-acc-user@example.com"),
					resource.TestCheckResourceAttr("superset_user.test", "active", "true"),
					resource.TestCheckResourceAttr("superset_user.test", "roles.#", "1"),
					resource.TestCheckResourceAttrSet("superset_user.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:            "superset_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "password"},
			},
			// Update: change last_name and email
			{
				Config: providerConfig() + `
data "superset_role" "gamma" {
  name = "Gamma"
}

resource "superset_user" "test" {
  username   = "tf-acc-user"
  first_name = "TFAcc"
  last_name  = "UpdatedUser"
  email      = "tf-acc-user-updated@example.com"
  active     = true
  roles      = [data.superset_role.gamma.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_user.test", "last_name", "UpdatedUser"),
					resource.TestCheckResourceAttr("superset_user.test", "email", "tf-acc-user-updated@example.com"),
				),
			},
		},
	})
}
