// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// rolePermissionsConfig renders a role, its permissions and a data source that reads the
// permissions back from Superset, so checks cover the API side and not only the state.
func rolePermissionsConfig(roleName string, perms [][2]string) string {
	var items strings.Builder
	for _, p := range perms {
		fmt.Fprintf(&items, "    { permission = %q, view_menu = %q },\n", p[0], p[1])
	}
	return providerConfig() + fmt.Sprintf(`
resource "superset_role" "test" {
  name = %q
}

resource "superset_role_permissions" "test" {
  role_name            = superset_role.test.name
  resource_permissions = [
%s  ]
}

data "superset_role_permissions" "test" {
  role_name  = superset_role.test.name
  depends_on = [superset_role_permissions.test]
}
`, roleName, items.String())
}

// TestAccRolePermissionsResource_Empty covers roles without permissions: create with an
// empty list, add a permission, then clear it again. Each step is followed by the
// framework's automatic plan, which fails if the empty list is stored as null.
func TestAccRolePermissionsResource_Empty(t *testing.T) {
	const roleName = "tf-acc-role-perms-empty"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: rolePermissionsConfig(roleName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", "0"),
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "permissions.#", "0"),
				),
			},
			{
				Config: rolePermissionsConfig(roleName, [][2]string{{"can_read", "Chart"}}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", "1"),
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "permissions.#", "1"),
				),
			},
			{
				Config: rolePermissionsConfig(roleName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", "0"),
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "permissions.#", "0"),
				),
			},
		},
	})
}

// manyPermissions spans several pages of /api/v1/security/permissions-resources/ (100 per
// page) and includes entries that unordered pagination used to skip on a real instance
// (Api.can_time_range, Api.can_query, SQLLab.*).
var manyPermissions = [][2]string{
	{"can_read", "Annotation"},
	{"can_time_range", "Api"},
	{"can_query", "Api"},
	{"can_query_form_data", "Api"},
	{"can_read", "Chart"},
	{"can_read", "CssTemplate"},
	{"can_read", "Dashboard"},
	{"can_view_chart_as_table", "Dashboard"},
	{"can_drill", "Dashboard"},
	{"can_read", "DashboardFilterStateRestApi"},
	{"menu_access", "Dashboards"},
	{"can_read", "Database"},
	{"menu_access", "Databases"},
	{"can_read", "Dataset"},
	{"can_get_drill_info", "Dataset"},
	{"can_external_metadata", "Datasource"},
	{"can_get", "Datasource"},
	{"can_external_metadata_by_name", "Datasource"},
	{"can_samples", "Datasource"},
	{"menu_access", "Home"},
	{"can_recent_activity", "Log"},
	{"can_get", "MenuApi"},
	{"can_read", "Query"},
	{"can_read", "SavedQuery"},
	{"can_read", "SQLLab"},
	{"can_execute_sql_query", "SQLLab"},
	{"can_dashboard_permalink", "Superset"},
	{"can_fetch_datasource_metadata", "Superset"},
	{"can_slice", "Superset"},
	{"can_log", "Superset"},
	{"can_explore_json", "Superset"},
	{"can_csv", "Superset"},
}

// TestAccRolePermissionsResource_ManyPermissions resolves permissions spread across the
// whole permission list, then removes one to exercise Update with a large set.
func TestAccRolePermissionsResource_ManyPermissions(t *testing.T) {
	const roleName = "tf-acc-role-perms-many"
	fewer := manyPermissions[:len(manyPermissions)-1]
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: rolePermissionsConfig(roleName, manyPermissions),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", strconv.Itoa(len(manyPermissions))),
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "permissions.#", strconv.Itoa(len(manyPermissions))),
					resource.TestCheckTypeSetElemNestedAttrs("data.superset_role_permissions.test", "permissions.*", map[string]string{
						"permission_name": "can_time_range",
						"view_menu_name":  "Api",
					}),
				),
			},
			{
				Config: rolePermissionsConfig(roleName, fewer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_role_permissions.test", "resource_permissions.#", strconv.Itoa(len(fewer))),
					resource.TestCheckResourceAttr("data.superset_role_permissions.test", "permissions.#", strconv.Itoa(len(fewer))),
				),
			},
		},
	})
}

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
