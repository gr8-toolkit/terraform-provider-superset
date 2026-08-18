terraform {
  required_providers {
    superset = {
      source = "svdimchenko/superset"
    }
  }
}

provider "superset" {
  host     = "http://localhost:8088"
  username = "admin"
  password = "admin"
}

# Create a CSS template with inline CSS
resource "superset_css_template" "example" {
  template_name = "Dashboard Global Styles"
  css           = <<-EOT
    .dashboard-component-chart-holder {
      background: #fff !important;
      border: 1px solid rgba(0, 0, 0, 0.1) !important;
      border-radius: 8px !important;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06) !important;
    }
  EOT
}

# Create a CSS template from a file
resource "superset_css_template" "from_file" {
  template_name = "Dashboard Styles From File"
  css           = file("${path.module}/dashboard.css")
}
