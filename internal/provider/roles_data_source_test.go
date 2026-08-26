// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRolesDataSource verifies the superset_roles data source returns at least
// the built-in "Admin" role that every fresh Superset installation ships with.
func TestAccRolesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
data "superset_roles" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Every fresh Superset ships with at least Admin, Public, Alpha, Gamma, sql_lab.
					resource.TestCheckResourceAttrSet("data.superset_roles.test", "roles.#"),
				),
			},
		},
	})
}
