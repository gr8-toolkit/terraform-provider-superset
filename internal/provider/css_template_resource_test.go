package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func TestAccCSSTemplateResource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock login
	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/security/login",
		httpmock.NewStringResponder(200, `{"access_token": "fake-token"}`))

	// Mock CSRF token
	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "fake-csrf"}`))

	// Mock FindCSSTemplatesByName (uniqueness check - no existing templates)
	httpmock.RegisterResponder("GET", `=~^http://superset-host/api/v1/css_template/\?q=.*`,
		httpmock.NewStringResponder(200, `{"result": [], "count": 0}`))

	// Mock Create
	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/css_template/",
		httpmock.NewStringResponder(201, `{"id": 1, "result": {"id": 1, "template_name": "Test Template", "css": "body { color: red; }"}}`))

	// Mock Read
	httpmock.RegisterResponder("GET", "http://superset-host/api/v1/css_template/1",
		httpmock.NewStringResponder(200, `{"result": {"id": 1, "template_name": "Test Template", "css": "body { color: red; }"}}`))

	// Mock Delete
	httpmock.RegisterResponder("DELETE", "http://superset-host/api/v1/css_template/1",
		httpmock.NewStringResponder(200, ``))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "superset_css_template" "test" {
  template_name = "Test Template"
  css           = "body { color: red; }"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_css_template.test", "template_name", "Test Template"),
					resource.TestCheckResourceAttr("superset_css_template.test", "css", "body { color: red; }"),
					resource.TestCheckResourceAttrSet("superset_css_template.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "superset_css_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
