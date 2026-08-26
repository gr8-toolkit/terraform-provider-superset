// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// providerConfig returns the HCL provider block using environment variables.
// Tests prepend this to every resource/data-source config they pass to resource.Test.
func providerConfig() string {
	host := os.Getenv("SUPERSET_HOST")
	username := os.Getenv("SUPERSET_USERNAME")
	password := os.Getenv("SUPERSET_PASSWORD")

	return fmt.Sprintf(`
provider "superset" {
  host     = %q
  username = %q
  password = %q
}
`, host, username, password)
}

// testAccProtoV6ProviderFactories wires the in-process provider for terraform-plugin-testing.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"superset": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that the required environment variables are set before any test runs.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if v := os.Getenv("SUPERSET_HOST"); v == "" {
		t.Fatal("SUPERSET_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("SUPERSET_USERNAME"); v == "" {
		t.Fatal("SUPERSET_USERNAME must be set for acceptance tests")
	}
	if v := os.Getenv("SUPERSET_PASSWORD"); v == "" {
		t.Fatal("SUPERSET_PASSWORD must be set for acceptance tests")
	}
}
