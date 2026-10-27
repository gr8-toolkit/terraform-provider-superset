// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const rolesListResponse = `{
	"result": [
		{"id": 1, "name": "Admin"},
		{"id": 4, "name": "Gamma"}
	]
}`

func TestGetRoleIDByName_Found(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles?q=(page_size:5000)",
		httpmock.NewStringResponder(200, rolesListResponse))

	id, err := c.GetRoleIDByName("Gamma")
	assert.NoError(t, err)
	assert.Equal(t, int64(4), id)
}

func TestGetRoleIDByName_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles?q=(page_size:5000)",
		httpmock.NewStringResponder(200, rolesListResponse))

	_, err := c.GetRoleIDByName("NoSuchRole")
	assert.Error(t, err)
}

func TestFetchRoles(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles?q=(page_size:5000)",
		httpmock.NewStringResponder(200, rolesListResponse))

	roles, err := c.FetchRoles()
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, int64(1), roles[0].ID)
	assert.Equal(t, "Admin", roles[0].Name)
}

func TestCreateRole(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	// GetRoleIDByName: role does not exist yet
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{"result": []}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/roles/",
		httpmock.NewStringResponder(201, `{"id": 99}`))

	id, err := c.CreateRole("NewRole")
	assert.NoError(t, err)
	assert.Equal(t, int64(99), id)
}

func TestCreateRole_AlreadyExists(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{"result": [{"id": 5, "name": "Existing"}]}`))

	id, err := c.CreateRole("Existing")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), id) // returns existing ID, no POST
}

func TestGetRole(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles/1",
		httpmock.NewStringResponder(200, `{"result": {"id": 1, "name": "Admin"}}`))

	role, err := c.GetRole(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), role.ID)
	assert.Equal(t, "Admin", role.Name)
}

func TestGetRole_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetRole(99)
	assert.Error(t, err)
}

func TestUpdateRole(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles/1",
		httpmock.NewStringResponder(200, `{"result": {"id": 1, "name": "OldName"}}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/security/roles/1",
		httpmock.NewStringResponder(200, `{"id": 1, "name": "NewName"}`))

	err := c.UpdateRole(1, "NewName")
	assert.NoError(t, err)
}

func TestUpdateRole_SameName_NoOp(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles/1",
		httpmock.NewStringResponder(200, `{"result": {"id": 1, "name": "Same"}}`))

	err := c.UpdateRole(1, "Same") // no PUT should fire
	assert.NoError(t, err)
}

func TestDeleteRole(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/security/roles/1",
		httpmock.NewStringResponder(204, ""))

	err := c.DeleteRole(1)
	assert.NoError(t, err)
}

func TestDeleteRole_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/security/roles/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteRole(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete role, status code: 404")
}

func TestGetRolePermissions(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/roles/1/permissions/",
		httpmock.NewStringResponder(200, `{"result": [
			{"id": 10, "permission_name": "can_read", "view_menu_name": "Chart"},
			{"id": 11, "permission_name": "can_read", "view_menu_name": "Dashboard"}
		]}`))

	perms, err := c.GetRolePermissions(1)
	assert.NoError(t, err)
	assert.Len(t, perms, 2)
	assert.Equal(t, int64(10), perms[0].ID)
	assert.Equal(t, "can_read", perms[0].PermissionName)
	assert.Equal(t, "Chart", perms[0].ViewMenuName)
}

// permissionsPage builds a permissions-resources page with n entries starting at firstID.
func permissionsPage(firstID, n int) string {
	items := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := firstID + i
		items = append(items, fmt.Sprintf(`{"id": %d, "permission": {"name": "can_p%d"}, "view_menu": {"name": "View%d"}}`, id, id, id))
	}
	return `{"result": [` + strings.Join(items, ",") + `]}`
}

func registerPermissionsPage(page int, body string) {
	httpmock.RegisterResponderWithQuery("GET", "http://test-host/api/v1/security/permissions-resources/",
		fmt.Sprintf("q=(order_column:id,order_direction:asc,page:%d,page_size:100)", page),
		httpmock.NewStringResponder(200, body))
}

func TestGetPermissionIDMap_OrderedPagination(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	registerPermissionsPage(0, permissionsPage(1, 100))
	registerPermissionsPage(1, permissionsPage(101, 30))

	ids, err := c.GetPermissionIDMap()
	assert.NoError(t, err)
	assert.Len(t, ids, 130)
	assert.Equal(t, int64(1), ids[PermissionKey("can_p1", "View1")])
	assert.Equal(t, int64(130), ids[PermissionKey("can_p130", "View130")])
	assert.Equal(t, 2, httpmock.GetTotalCallCount())
}

func TestGetPermissionIDByNameAndView(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	registerPermissionsPage(0, permissionsPage(1, 5))

	id, err := c.GetPermissionIDByNameAndView("can_p3", "View3")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), id)

	_, err = c.GetPermissionIDByNameAndView("can_time_range", "Api")
	assert.Error(t, err)
}

func TestGetPermissionViewMenuIDs_SkipsMissing(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	registerPermissionsPage(0, permissionsPage(1, 5))

	ids, err := c.GetPermissionViewMenuIDs([]map[string]string{
		{"permission": "can_p2", "view_menu": "View2"},
		{"permission": "can_missing", "view_menu": "Nope"},
	})
	assert.NoError(t, err)
	assert.Equal(t, []int64{2}, ids)
}

func TestUpdateRolePermissions_NilSendsEmptyList(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	var sent string
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/roles/1/permissions",
		func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			sent = string(body)
			return httpmock.NewStringResponse(200, `{}`), nil
		})

	err := c.UpdateRolePermissions(1, nil)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"permission_view_menu_ids": []}`, sent)
}

func TestClearRolePermissions(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/roles/1/permissions",
		httpmock.NewStringResponder(200, `{}`))

	err := c.ClearRolePermissions(1)
	assert.NoError(t, err)
}
