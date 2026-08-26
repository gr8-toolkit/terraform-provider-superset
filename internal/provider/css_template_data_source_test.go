// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCSSTemplateDataSource creates a CSS template then looks it up via the
// superset_css_template data source.
func TestAccCSSTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "superset_css_template" "ds_test" {
  template_name = "tf-acc-css-template-ds"
  css           = ".chart { border: 1px solid #ccc; }"
}

data "superset_css_template" "test" {
  name       = superset_css_template.ds_test.template_name
  depends_on = [superset_css_template.ds_test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.superset_css_template.test", "name", "tf-acc-css-template-ds"),
					resource.TestCheckResourceAttr("data.superset_css_template.test", "template_name", "tf-acc-css-template-ds"),
					resource.TestCheckResourceAttr("data.superset_css_template.test", "css", ".chart { border: 1px solid #ccc; }"),
					resource.TestCheckResourceAttrSet("data.superset_css_template.test", "id"),
				),
			},
		},
	})
}
