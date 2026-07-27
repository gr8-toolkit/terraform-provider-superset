resource "superset_dataset_import" "example" {
  source_dir      = "${path.module}/dashboards/my_dashboard"
  force_overwrite = true

  database_secrets = {
    "db-uuid" = var.db_password
  }

  database_overrides = {
    "db-uuid" = jsonencode({
      sqlalchemy_uri = "postgresql://user:pass@host:5432/mydb"
    })
  }

  # Exclude files from hashing to avoid spurious diffs
  skip_files = [".*terragrunt.*", "\\.terraform\\.lock\\.hcl"]
}
