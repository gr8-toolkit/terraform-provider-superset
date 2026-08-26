// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// ── importViaEndpoint (covers ImportDashboard / ImportDataset / ImportChart) ──

func TestImportDashboard(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/import/",
		httpmock.NewStringResponder(200, `{"message": "OK"}`))

	err := c.ImportDashboard([]byte("fake-zip"), true, "")
	assert.NoError(t, err)
}

func TestImportDashboard_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/import/",
		httpmock.NewStringResponder(400, `{"message": "bad zip"}`))

	err := c.ImportDashboard([]byte("bad"), false, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "import failed")
}

func TestImportDataset(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dataset/import/",
		httpmock.NewStringResponder(200, `{"message": "OK"}`))

	err := c.ImportDataset([]byte("fake-zip"), true, "")
	assert.NoError(t, err)
}

func TestImportChart(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/chart/import/",
		httpmock.NewStringResponder(200, `{"message": "OK"}`))

	err := c.ImportChart([]byte("fake-zip"), true, "")
	assert.NoError(t, err)
}

// ── Dashboard lookup helpers ──────────────────────────────────────────────────

func TestGetDashboardIDByUUID(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET",
		`=~^http://test-host/api/v1/dashboard/\?q=`,
		httpmock.NewStringResponder(200, `{"result": [{"id": 10}]}`))

	id, err := c.GetDashboardIDByUUID("some-uuid")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), id)
}

func TestGetDashboardIDByUUID_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET",
		`=~^http://test-host/api/v1/dashboard/\?q=`,
		httpmock.NewStringResponder(200, `{"result": []}`))

	_, err := c.GetDashboardIDByUUID("missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDashboardExistsByID_True(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/10",
		httpmock.NewStringResponder(200, `{"result": {"id": 10}}`))

	exists, err := c.DashboardExistsByID(10)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDashboardExistsByID_False(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	exists, err := c.DashboardExistsByID(99)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteDashboard(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dashboard/10",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteDashboard(10)
	assert.NoError(t, err)
}
