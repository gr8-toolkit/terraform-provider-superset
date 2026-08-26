resource "superset_saved_query" "example" {
  database_id = superset_database.example.id
  label       = "Count active users"
  sql         = "SELECT COUNT(*) FROM ab_user WHERE active = true"
  schema      = "public"
}
