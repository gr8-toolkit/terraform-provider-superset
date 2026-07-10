resource "superset_chart_import" "example" {
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
}
