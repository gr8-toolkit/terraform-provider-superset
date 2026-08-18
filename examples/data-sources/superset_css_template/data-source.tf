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

# Look up an existing CSS template by name
data "superset_css_template" "default_styles" {
  name = "Dashboard Global Styles"
}

# Use the CSS content from the template
output "template_id" {
  value = data.superset_css_template.default_styles.id
}

output "template_css" {
  value = data.superset_css_template.default_styles.css
}
