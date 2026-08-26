// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetDashboardEmbedded(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/10/embedded",
		httpmock.NewStringResponder(200, `{"result": {"uuid": "embed-uuid", "allowed_domains": ["example.com"]}}`))

	emb, err := c.GetDashboardEmbedded(10)
	assert.NoError(t, err)
	assert.NotNil(t, emb)
	assert.Equal(t, "embed-uuid", emb.UUID)
	assert.Equal(t, []string{"example.com"}, emb.AllowedDomains)
}

func TestGetDashboardEmbedded_NotFound_ReturnsNil(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dashboard/10/embedded",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	emb, err := c.GetDashboardEmbedded(10)
	assert.NoError(t, err)
	assert.Nil(t, emb) // 404 treated as "not configured"
}

func TestCreateDashboardEmbedded(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/10/embedded",
		httpmock.NewStringResponder(200, `{"result": {"uuid": "new-uuid", "allowed_domains": ["*.corp.com"]}}`))

	emb, err := c.CreateDashboardEmbedded(10, []string{"*.corp.com"})
	assert.NoError(t, err)
	assert.Equal(t, "new-uuid", emb.UUID)
	assert.Equal(t, []string{"*.corp.com"}, emb.AllowedDomains)
}

func TestCreateDashboardEmbedded_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dashboard/99/embedded",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.CreateDashboardEmbedded(99, []string{})
	assert.Error(t, err)
}

func TestDeleteDashboardEmbedded(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dashboard/10/embedded",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteDashboardEmbedded(10)
	assert.NoError(t, err)
}

func TestDeleteDashboardEmbedded_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dashboard/99/embedded",
		httpmock.NewStringResponder(500, `{"message": "server error"}`))

	err := c.DeleteDashboardEmbedded(99)
	assert.Error(t, err)
}
