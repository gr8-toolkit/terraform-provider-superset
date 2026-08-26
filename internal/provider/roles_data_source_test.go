// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRolesDataSource verifies the superset_roles data source returns the
// built-in roles that every fresh Superset installation ships with.
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
					// Every fresh Superset ships with Admin, Public, Alpha, Gamma, sql_lab — at least 5.
					resource.TestCheckResourceAttrSet("data.superset_roles.test", "roles.#"),
					// Verify the Admin role is actually present by name.
					resource.TestCheckTypeSetElemNestedAttrs(
						"data.superset_roles.test", "roles.*",
						map[string]string{"name": "Admin"},
					),
					// Verify Gamma is present — used by other tests as a known role.
					resource.TestCheckTypeSetElemNestedAttrs(
						"data.superset_roles.test", "roles.*",
						map[string]string{"name": "Gamma"},
					),
				),
			},
		},
	})
}
