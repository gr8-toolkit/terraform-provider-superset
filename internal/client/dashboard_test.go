// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const dashGetResponse = `{
	"result": {
		"id": 10,
		"uuid": "dash-uuid",
		"dashboard_title": "Sales Dashboard",
		"slug": "sales",
		"css": "",
		"published": true,
		"position_json": "{}",
		"json_metadata": "{}",
		"url": "/superset/dashboard/sales/"
	}
}`

func TestCreateDashboard(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/",
		httpmock.NewStringResponder(201, `{"id": 10, "result": {"dashboard_title": "Sales Dashboard", "published": false}}`))

	dash, err := c.CreateDashboard(map[string]interface{}{"dashboard_title": "Sales Dashboard", "published": false})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), dash.ID)
	assert.Equal(t, "Sales Dashboard", dash.DashboardTitle)
}

func TestCreateDashboard_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateDashboard(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetDashboard(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/10",
		httpmock.NewStringResponder(200, dashGetResponse))

	dash, err := c.GetDashboard(10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), dash.ID)
	assert.Equal(t, "dash-uuid", dash.UUID)
	assert.Equal(t, "Sales Dashboard", dash.DashboardTitle)
	assert.Equal(t, "sales", dash.Slug)
	assert.True(t, dash.Published)
}

func TestGetDashboard_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetDashboard(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}

func TestUpdateDashboard(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/dashboard/10",
		httpmock.NewStringResponder(200, `{"result": {"id": 10, "dashboard_title": "Updated", "published": true}}`))

	dash, err := c.UpdateDashboard(10, map[string]interface{}{"dashboard_title": "Updated", "published": true})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), dash.ID)
}

func TestDeleteDashboardByID(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dashboard/10",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteDashboardByID(10)
	assert.NoError(t, err)
}

func TestDeleteDashboardByID_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dashboard/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteDashboardByID(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}
