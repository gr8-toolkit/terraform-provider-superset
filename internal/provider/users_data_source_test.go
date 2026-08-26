// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUsersDataSource seeds one user then reads the full list via the
// superset_users data source.
func TestAccUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
data "superset_role" "gamma" {
  name = "Gamma"
}

resource "superset_user" "ds_seed" {
  username   = "tf-acc-users-ds"
  first_name = "TFAcc"
  last_name  = "UsersDS"
  email      = "tf-acc-users-ds@example.com"
  password   = "Acc3ptanceT3st!"
  active     = true
  roles      = [data.superset_role.gamma.id]
}

data "superset_users" "test" {
  depends_on = [superset_user.ds_seed]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// At minimum the seeded user plus the built-in admin should be present.
					resource.TestCheckResourceAttrSet("data.superset_users.test", "users.#"),
				),
			},
		},
	})
}
