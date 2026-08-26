// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAnnotationLayerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: providerConfig() + `
resource "superset_annotation_layer" "test" {
  name        = "tf-acc-annotation-layer"
  description = "Created by acceptance test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "name", "tf-acc-annotation-layer"),
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "description", "Created by acceptance test"),
					resource.TestCheckResourceAttrSet("superset_annotation_layer.test", "id"),
				),
			},
			// ImportState
			{
				ResourceName:      "superset_annotation_layer.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: providerConfig() + `
resource "superset_annotation_layer" "test" {
  name        = "tf-acc-annotation-layer"
  description = "Updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("superset_annotation_layer.test", "description", "Updated description"),
				),
			},
		},
	})
}
