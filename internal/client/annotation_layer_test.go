// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateAnnotationLayer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/annotation_layer/",
		httpmock.NewStringResponder(201, `{"id": 7, "result": {"id": 7, "name": "Deployments", "descr": "Deploy events"}}`))

	al, err := c.CreateAnnotationLayer("Deployments", "Deploy events")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), al.ID)
	assert.Equal(t, "Deployments", al.Name)
	assert.Equal(t, "Deploy events", al.Description)
}

func TestCreateAnnotationLayer_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/annotation_layer/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateAnnotationLayer("Bad", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetAnnotationLayer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/annotation_layer/7",
		httpmock.NewStringResponder(200, `{"result": {"id": 7, "name": "Deployments", "descr": "Deploy events"}}`))

	al, err := c.GetAnnotationLayer(7)
	assert.NoError(t, err)
	assert.Equal(t, int64(7), al.ID)
	assert.Equal(t, "Deployments", al.Name)
	assert.Equal(t, "Deploy events", al.Description)
}

func TestGetAnnotationLayer_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/annotation_layer/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetAnnotationLayer(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}

func TestUpdateAnnotationLayer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/annotation_layer/7",
		httpmock.NewStringResponder(200, `{"result": {"id": 7, "name": "Updated Layer", "descr": "New description"}}`))

	al, err := c.UpdateAnnotationLayer(7, "Updated Layer", "New description")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Layer", al.Name)
	assert.Equal(t, "New description", al.Description)
}

func TestDeleteAnnotationLayer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/annotation_layer/7",
		httpmock.NewStringResponder(200, ""))

	err := c.DeleteAnnotationLayer(7)
	assert.NoError(t, err)
}

func TestDeleteAnnotationLayer_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/annotation_layer/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteAnnotationLayer(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 404")
}
