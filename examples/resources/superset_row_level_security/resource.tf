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

# Fetch role by name
data "superset_role" "analyst" {
  name = "Analyst"
}

data "superset_role" "manager" {
  name = "Manager"
}

resource "superset_dataset" "example" {
  database_name = "source"
  schema        = "database"
  table_name    = "table"
}

resource "superset_row_level_security" "example" {
  name        = "User Data Filter"
  tables      = [superset_dataset.example.id]
  clause      = "user_id = '{{ current_user_id() }}'"
  role_ids    = [data.superset_role.analyst.id, data.superset_role.manager.id]
  group_key   = "department"
  filter_type = "Regular"
  description = "User-level data access filter"
}
