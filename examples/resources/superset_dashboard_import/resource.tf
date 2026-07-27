resource "superset_dashboard_import" "example" {
  source_dir      = "${path.module}/dashboards/my_dashboard"
  force_overwrite = true

  database_secrets = {
    "db-uuid" = var.db_password
  }

  # Override any database YAML fields (including nested) by UUID.
  # Value is a JSON-encoded object that gets deep-merged into the YAML.
  database_overrides = {
    "db-uuid" = jsonencode({
      sqlalchemy_uri = "postgresql://user:pass@host:5432/mydb"
      extra = {
        schemas_allowed_for_file_upload = ["public"]
        cost_estimate_enabled           = false
      }
    })
  }

  # Role IDs to assign to the dashboard. Applied after every create/update.
  roles = [superset_role.analytics.id, superset_role.viewers.id]

  # Exclude files from hashing to avoid spurious diffs
  skip_files = [".*terragrunt.*", "\\.terraform\\.lock\\.hcl"]
}
