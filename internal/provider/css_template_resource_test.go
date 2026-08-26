// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCSSTemplateResource covers create, read, import, and update.
func TestAccCSSTemplateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_css_template" "test" {
  template_name = "tf-acc-css-template"
  css           = "body { color: red; }"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_css_template.test", "template_name", "tf-acc-css-template"),
					resource.TestCheckResourceAttr("superset_css_template.test", "css", "body { color: red; }"),
					resource.TestCheckResourceAttrSet("superset_css_template.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_css_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update: change CSS content
			{
				Config: providerConfig() + `
resource "superset_css_template" "test" {
  template_name = "tf-acc-css-template"
  css           = "body { color: blue; font-size: 14px; }"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_css_template.test", "css", "body { color: blue; font-size: 14px; }"),
				),
			},
		},
	})
}
