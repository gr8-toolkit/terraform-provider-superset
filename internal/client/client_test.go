// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// ── NewClient / authenticate ──────────────────────────────────────────────────

func TestNewClient_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/login",
		httpmock.NewStringResponder(200, `{"access_token": "my-token"}`))

	c, err := NewClient("http://test-host", "admin", "admin", "db")
	assert.NoError(t, err)
	assert.Equal(t, "my-token", c.Token)
}

func TestNewClient_AuthFailure(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/login",
		httpmock.NewStringResponder(401, `{"message": "Unauthorized"}`))

	c, err := NewClient("http://test-host", "admin", "wrong", "db")
	assert.Error(t, err)
	assert.Nil(t, c)
	assert.Contains(t, err.Error(), "failed to authenticate")
}

// ── GetCSRFToken ──────────────────────────────────────────────────────────────

func TestGetCSRFToken(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf-abc"}`))

	token, _, err := c.GetCSRFToken()
	assert.NoError(t, err)
	assert.Equal(t, "csrf-abc", token)
}

func TestGetCSRFToken_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(500, `{"message": "error"}`))

	_, _, err := c.GetCSRFToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get CSRF token")
}

// ── ClearGlobalDatabaseCache ──────────────────────────────────────────────────

func TestClearGlobalDatabaseCache(t *testing.T) {
	globalDatabasesCache = []map[string]interface{}{{"id": float64(1)}}
	ClearGlobalDatabaseCache()
	assert.Nil(t, globalDatabasesCache)
}
