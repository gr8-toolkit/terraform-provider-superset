// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const savedQueryGetResponse = `{
	"result": {
		"id": 3,
		"db_id": 1,
		"database": {"id": 1, "database_name": "PostgreSQL"},
		"label": "Count users",
		"description": null,
		"catalog": null,
		"schema": "public",
		"sql": "SELECT COUNT(*) FROM ab_user",
		"template_parameters": null,
		"extra_json": null
	}
}`

func TestCreateSavedQuery(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/saved_query/",
		httpmock.NewStringResponder(201, `{"id": 3, "result": {"id": 3, "db_id": 1, "label": "Count users", "sql": "SELECT COUNT(*) FROM ab_user"}}`))

	sq, err := c.CreateSavedQuery(map[string]interface{}{
		"db_id": int64(1),
		"label": "Count users",
		"sql":   "SELECT COUNT(*) FROM ab_user",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), sq.ID)
	assert.Equal(t, "Count users", sq.Label)
}

func TestCreateSavedQuery_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/saved_query/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateSavedQuery(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetSavedQuery(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/saved_query/3",
		httpmock.NewStringResponder(200, savedQueryGetResponse))

	sq, err := c.GetSavedQuery(3)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), sq.ID)
	assert.Equal(t, int64(1), sq.DatabaseID)
	assert.Equal(t, "PostgreSQL", sq.DatabaseName)
	assert.Equal(t, "Count users", sq.Label)
	assert.Equal(t, "public", sq.Schema)
	assert.Equal(t, "SELECT COUNT(*) FROM ab_user", sq.SQL)
}

func TestGetSavedQuery_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/saved_query/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetSavedQuery(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}

func TestUpdateSavedQuery(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/saved_query/3",
		httpmock.NewStringResponder(200, `{"result": {"id": 3, "db_id": 1, "database": {"id": 1, "database_name": "PostgreSQL"}, "label": "Updated", "sql": "SELECT 2"}}`))

	sq, err := c.UpdateSavedQuery(3, map[string]interface{}{"db_id": int64(1), "label": "Updated", "sql": "SELECT 2"})
	assert.NoError(t, err)
	assert.Equal(t, "Updated", sq.Label)
}

func TestDeleteSavedQuery(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/saved_query/3",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteSavedQuery(3)
	assert.NoError(t, err)
}

func TestDeleteSavedQuery_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/saved_query/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteSavedQuery(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}

func TestParseSavedQueryResponse_NullFields(t *testing.T) {
	// Verify that null API fields map to empty strings, not panics.
	raw := savedQueryResponse{
		ID:         5,
		DatabaseID: 2,
		Database: &struct {
			ID           int64  `json:"id"`
			DatabaseName string `json:"database_name"`
		}{ID: 2, DatabaseName: "mydb"},
		Label: "q",
		SQL:   "SELECT 1",
		// Description, Catalog, Schema, TemplateParameters, ExtraJSON all nil
	}
	sq := parseSavedQueryResponse(raw)
	assert.Equal(t, int64(5), sq.ID)
	assert.Equal(t, "mydb", sq.DatabaseName)
	assert.Equal(t, "", sq.Description)
	assert.Equal(t, "", sq.Catalog)
	assert.Equal(t, "", sq.Schema)
	assert.Equal(t, "", sq.TemplateParameters)
	assert.Equal(t, "", sq.ExtraJSON)
}
