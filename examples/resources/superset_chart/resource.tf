resource "superset_chart" "example" {
  slice_name    = "Example Table"
  viz_type      = "table"
  datasource_id = superset_dataset.example.id
  params        = jsonencode({ metrics = ["count"] })
}
