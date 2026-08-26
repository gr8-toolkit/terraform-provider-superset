// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const dbListResponse = `{
	"result": [
		{"id": 1, "database_name": "PostgreSQL"},
		{"id": 2, "database_name": "MySQL"}
	]
}`

func TestGetAllDatabases(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, dbListResponse))

	dbs, err := c.GetAllDatabases()
	assert.NoError(t, err)
	assert.Len(t, dbs, 2)
	assert.Equal(t, "PostgreSQL", dbs[0]["database_name"])
}

func TestGetAllDatabases_UsesCache(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, dbListResponse))

	_, _ = c.GetAllDatabases()      // prime cache
	dbs, err := c.GetAllDatabases() // should use cache, no second HTTP call
	assert.NoError(t, err)
	assert.Len(t, dbs, 2)
	assert.Equal(t, 1, httpmock.GetTotalCallCount()) // only one real call
}

func TestGetAllDatabases_Error(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(500, `{}`))

	_, err := c.GetAllDatabases()
	assert.Error(t, err)
}

func TestGetDatabaseConnectionByID(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/1/connection",
		httpmock.NewStringResponder(200, `{"result": {"database_name": "PostgreSQL", "sqlalchemy_uri": "postgresql://..."}}`))

	detail, err := c.GetDatabaseConnectionByID(1)
	assert.NoError(t, err)
	assert.NotNil(t, detail)
}

func TestCreateDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{"id": 10, "result": {"database_name": "NewDB"}}`))

	result, err := c.CreateDatabase(map[string]interface{}{"database_name": "NewDB"})
	assert.NoError(t, err)
	assert.Equal(t, float64(10), result["id"])
}

func TestCreateDatabase_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/database/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateDatabase(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestUpdateDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/database/10",
		httpmock.NewStringResponder(200, `{"id": 10, "result": {"database_name": "UpdatedDB"}}`))

	result, err := c.UpdateDatabase(10, map[string]interface{}{"database_name": "UpdatedDB"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/database/10",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteDatabase(10)
	assert.NoError(t, err)
}

func TestGetDatabaseIDByName(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, dbListResponse))

	id, err := c.GetDatabaseIDByName("MySQL")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), id)
}

func TestGetDatabaseIDByName_NotFound(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, dbListResponse))

	_, err := c.GetDatabaseIDByName("Nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetDatabaseNameByID(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, dbListResponse))

	name, err := c.GetDatabaseNameByID(1)
	assert.NoError(t, err)
	assert.Equal(t, "PostgreSQL", name)
}
