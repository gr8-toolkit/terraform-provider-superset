package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateCSSTemplate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("POST", "http://test-host/api/v1/css_template/",
		httpmock.NewStringResponder(201, `{"id": 42, "result": {"id": 42, "template_name": "Test", "css": "body {}"}}`))

	tmpl, err := client.CreateCSSTemplate("Test", "body {}")

	assert.NoError(t, err)
	assert.Equal(t, 42, tmpl.ID)
	assert.Equal(t, "Test", tmpl.TemplateName)
	assert.Equal(t, "body {}", tmpl.CSS)
}

func TestGetCSSTemplate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/css_template/42",
		httpmock.NewStringResponder(200, `{"result": {"id": 42, "template_name": "Test", "css": "body {}"}}`))

	tmpl, err := client.GetCSSTemplate(42)

	assert.NoError(t, err)
	assert.Equal(t, 42, tmpl.ID)
	assert.Equal(t, "Test", tmpl.TemplateName)
	assert.Equal(t, "body {}", tmpl.CSS)
}

func TestGetCSSTemplate_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/css_template/999",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	tmpl, err := client.GetCSSTemplate(999)

	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateCSSTemplate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/css_template/42",
		httpmock.NewStringResponder(200, `{"result": {"id": 42, "template_name": "Updated", "css": "body { color: red; }"}}`))

	tmpl, err := client.UpdateCSSTemplate(42, "Updated", "body { color: red; }")

	assert.NoError(t, err)
	assert.Equal(t, 42, tmpl.ID)
	assert.Equal(t, "Updated", tmpl.TemplateName)
	assert.Equal(t, "body { color: red; }", tmpl.CSS)
}

func TestUpdateCSSTemplate_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/css_template/999",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	tmpl, err := client.UpdateCSSTemplate(999, "Test", "body {}")

	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteCSSTemplate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/css_template/42",
		httpmock.NewStringResponder(200, ``))

	err := client.DeleteCSSTemplate(42)

	assert.NoError(t, err)
}

func TestDeleteCSSTemplate_NotFound_IsSuccess(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/css_template/999",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := client.DeleteCSSTemplate(999)

	assert.NoError(t, err) // 404 on delete is treated as success
}

func TestFindCSSTemplateByName(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", `=~^http://test-host/api/v1/css_template/\?q=.*`,
		httpmock.NewStringResponder(200, `{"result": [{"id": 7, "template_name": "Dashboard Styles", "css": ".dash {}"}], "count": 1}`))

	tmpl, err := client.FindCSSTemplateByName("Dashboard Styles")

	assert.NoError(t, err)
	assert.Equal(t, 7, tmpl.ID)
	assert.Equal(t, "Dashboard Styles", tmpl.TemplateName)
	assert.Equal(t, ".dash {}", tmpl.CSS)
}

func TestFindCSSTemplateByName_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", `=~^http://test-host/api/v1/css_template/\?q=.*`,
		httpmock.NewStringResponder(200, `{"result": [], "count": 0}`))

	tmpl, err := client.FindCSSTemplateByName("NonExistent")

	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindCSSTemplateByName_Ambiguous(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", `=~^http://test-host/api/v1/css_template/\?q=.*`,
		httpmock.NewStringResponder(200, `{"result": [{"id": 1, "template_name": "Dup", "css": "a"}, {"id": 2, "template_name": "Dup", "css": "b"}], "count": 2}`))

	tmpl, err := client.FindCSSTemplateByName("Dup")

	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "multiple")
}
