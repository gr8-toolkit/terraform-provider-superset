// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetAllDatasets(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dataset/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{
			"result": [
				{"id": 1, "table_name": "orders"},
				{"id": 2, "table_name": "users"}
			]
		}`))

	datasets, err := c.GetAllDatasets()
	assert.NoError(t, err)
	assert.Len(t, datasets, 2)
	assert.Equal(t, "orders", datasets[0]["table_name"])
}

func TestCreateDataset(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dataset/",
		httpmock.NewStringResponder(201, `{"id": 42, "table_name": "orders"}`))

	result, err := c.CreateDataset(DatasetRequest{TableName: "orders", Database: 1, Schema: "public"})
	assert.NoError(t, err)
	assert.Equal(t, float64(42), (*result)["id"])
}

func TestCreateDataset_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/dataset/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateDataset(DatasetRequest{TableName: "bad"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetDataset(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dataset/42",
		httpmock.NewStringResponder(200, `{"result": {"id": 42, "table_name": "orders", "schema": "public"}}`))

	result, err := c.GetDataset(42)
	assert.NoError(t, err)
	assert.Equal(t, "orders", (*result)["table_name"])
}

func TestGetDataset_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/dataset/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetDataset(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateDataset(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/dataset/42",
		httpmock.NewStringResponder(200, `{}`))

	err := c.UpdateDataset(42, "new_table", "new_schema", "")
	assert.NoError(t, err)
}

func TestDeleteDataset(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dataset/42",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteDataset(42)
	assert.NoError(t, err)
}

func TestDeleteDataset_NotFound_IsSuccess(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/dataset/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteDataset(99)
	assert.NoError(t, err) // 404 treated as success
}

func TestGetDatasetIDByUUID(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET",
		`=~^http://test-host/api/v1/dataset/\?q=`,
		httpmock.NewStringResponder(200, `{"result": [{"id": 7}]}`))

	id, err := c.GetDatasetIDByUUID("some-uuid")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), id)
}

func TestGetDatasetIDByUUID_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET",
		`=~^http://test-host/api/v1/dataset/\?q=`,
		httpmock.NewStringResponder(200, `{"result": []}`))

	id, err := c.GetDatasetIDByUUID("missing-uuid")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), id)
}
