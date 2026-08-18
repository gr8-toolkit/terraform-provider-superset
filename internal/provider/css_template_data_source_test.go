package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func TestAccCSSTemplateDataSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock login
	httpmock.RegisterResponder("POST", "http://superset-host/api/v1/security/login",
		httpmock.NewStringResponder(200, `{"access_token": "fake-token"}`))

	// Mock FindCSSTemplateByName via list endpoint
	httpmock.RegisterResponder("GET", `=~^http://superset-host/api/v1/css_template/\?q=.*`,
		httpmock.NewStringResponder(200, `{"result": [{"id": 5, "template_name": "Dashboard Styles", "css": ".chart { border: 1px solid #ccc; }"}], "count": 1}`))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "superset_css_template" "test" {
  name = "Dashboard Styles"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.superset_css_template.test", "name", "Dashboard Styles"),
					resource.TestCheckResourceAttr("data.superset_css_template.test", "template_name", "Dashboard Styles"),
					resource.TestCheckResourceAttr("data.superset_css_template.test", "css", ".chart { border: 1px solid #ccc; }"),
					resource.TestCheckResourceAttrSet("data.superset_css_template.test", "id"),
				),
			},
		},
	})
}
