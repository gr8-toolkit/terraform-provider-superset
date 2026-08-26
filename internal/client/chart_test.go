// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const chartGetResponse = `{
	"result": {
		"id": 42,
		"uuid": "aaaa-bbbb",
		"slice_name": "My Chart",
		"viz_type": "table",
		"datasource_id": 1,
		"datasource_type": "table",
		"datasource_name_text": "public.orders",
		"params": "{\"metrics\":[\"count\"]}",
		"query_context": null,
		"cache_timeout": null,
		"url": "/explore/?slice_id=42"
	}
}`

func TestCreateChart(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/chart/",
		httpmock.NewStringResponder(201, `{"id": 42, "result": {"slice_name": "My Chart", "viz_type": "table", "datasource_id": 1, "datasource_type": "table", "params": "{}"}}`))

	ch, err := c.CreateChart(map[string]interface{}{
		"slice_name":      "My Chart",
		"viz_type":        "table",
		"datasource_id":   int64(1),
		"datasource_type": "table",
		"params":          "{}",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(42), ch.ID)
	assert.Equal(t, "My Chart", ch.SliceName)
}

func TestCreateChart_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/chart/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateChart(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetChart(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/chart/42",
		httpmock.NewStringResponder(200, chartGetResponse))

	ch, err := c.GetChart(42)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), ch.ID)
	assert.Equal(t, "aaaa-bbbb", ch.UUID)
	assert.Equal(t, "My Chart", ch.SliceName)
	assert.Equal(t, "table", ch.VizType)
	assert.Equal(t, "/explore/?slice_id=42", ch.URL)
}

func TestGetChart_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/chart/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetChart(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}

func TestUpdateChart(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/chart/42",
		httpmock.NewStringResponder(200, `{"result": {"id": 42, "slice_name": "Updated Chart", "viz_type": "bar", "datasource_id": 1, "datasource_type": "table", "params": "{}"}}`))

	ch, err := c.UpdateChart(42, map[string]interface{}{"slice_name": "Updated Chart", "viz_type": "bar"})
	assert.NoError(t, err)
	assert.Equal(t, int64(42), ch.ID)
	assert.Equal(t, "Updated Chart", ch.SliceName)
}

func TestUpdateChart_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/chart/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.UpdateChart(99, map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}
