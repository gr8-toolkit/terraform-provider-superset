terraform {
  required_providers {
    superset = {
      source = "gr8-toolkit/superset"
    }
  }
}

provider "superset" {
  host     = "http://localhost:8088"
  username = "admin"
  password = "admin"
}

# Fetch a role by name
data "superset_role" "analyst" {
  name = "Analyst"
}

# Use the role ID in a resource
resource "superset_row_level_security" "example" {
  name        = "User Data Filter"
  tables      = [1]
  clause      = "user_id = '{{ current_user_id() }}'"
  role_ids    = [data.superset_role.analyst.id]
  group_key   = "department"
  filter_type = "Regular"
  description = "User-level data access filter"
}

# Output the role ID
output "analyst_role_id" {
  value = data.superset_role.analyst.id
}
