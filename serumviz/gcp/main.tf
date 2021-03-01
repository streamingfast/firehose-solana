terraform {
  # This module is now only being testd with Terraform 0.13.x. However, to make upgrading easier, we are setting
  # 0.12.26 as the minimum version, as that version added support for required_providers with source URLs, making it
  # forwards compatible with 0.13.x code.
  required_version = ">= 0.12.26"

  backend "gcs" {
    bucket = "eoscanada-terraform-state"
    prefix  = "serumviz"
  }
}

provider "google" {
  version      = "3.53.0"
  access_token = var.gcp_access_token
  region       = var.region
  zone         = var.region_zone

  scopes = [
    "https://www.googleapis.com/auth/compute",
    "https://www.googleapis.com/auth/cloud-platform",
    "https://www.googleapis.com/auth/userinfo.email",
  ]
}

resource "google_bigquery_dataset" "serum" {
  dataset_id                  = "serum"
  friendly_name               = "serum"
  description                 = "serum events"
  location                    = "US"
  project                     = var.gcp_project
}

resource "google_bigquery_table" "fills" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "fills"
  project                     = var.gcp_project

  time_partitioning {
    field = "timestamp"
    type = "DAY"
  }

  schema = file("${path.module}/../schemas/v1/fills.json")
}

resource "google_bigquery_table" "orders" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "orders"
  project                     = var.gcp_project

  time_partitioning {
    field = "timestamp"
    type = "DAY"
  }

  schema = file("${path.module}/../schemas/v1/orders.json")
}

resource "google_bigquery_table" "processed_files" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "processed_files"
  project                     = var.gcp_project
  schema = file("${path.module}/../schemas/v1/processed_files.json")
}

resource "google_bigquery_table" "traders" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "traders"
  project                     = var.gcp_project
  schema = file("${path.module}/../schemas/v1/traders.json")
}

resource "google_bigquery_table" "markets" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "markets"
  project                     = var.gcp_project

  external_data_configuration {
    autodetect    = true
    source_format = "NEWLINE_DELIMITED_JSON"

    source_uris = [
      "gs://staging.dfuseio-global.appspot.com/sol-markets/sol-mainnet-v1.jsonl",
    ]
  }
}

resource "google_bigquery_table" "tokens" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "tokens"
  project                     = var.gcp_project

  external_data_configuration {
    autodetect    = true
    source_format = "NEWLINE_DELIMITED_JSON"

    source_uris = [
      "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl",
    ]
  }

}


//****************************************************************
//          Slot Timestamp View
//****************************************************************

data "template_file" "slot_timestamp_query" {
  template = file("${path.module}/queries/slot_timestamp.sql")
  vars = {
    dataset = google_bigquery_dataset.serum.dataset_id
  }
}

resource "google_bigquery_table" "slot_timestamp" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "slot_timestamp"
  project     = var.gcp_project

  view {
    query = data.template_file.slot_timestamp_query.rendered
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.fills]
}

//****************************************************************
//          Priced Fills View
//****************************************************************
data "template_file" "priced_fills_query" {
  template = file("${path.module}/queries/priced_fills.sql")
  vars = {
    dataset = google_bigquery_dataset.serum.dataset_id
  }
}

resource "google_bigquery_table" "priced_fills" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "priced_fills"
  project                     = var.gcp_project

  view {
    query = data.template_file.priced_fills_query.rendered
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.fills]
}


//****************************************************************
//          USD Priced Fills View
//****************************************************************
data "template_file" "usd_priced_fills_query" {
  template = file("${path.module}/queries/usd_priced_fills.sql")
  vars = {
    dataset = google_bigquery_dataset.serum.dataset_id
  }
}

resource "google_bigquery_table" "usd_priced_fills" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "usd_priced_fills"
  project                     = var.gcp_project

  view {
    query = data.template_file.usd_priced_fills_query.rendered
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.priced_fills]
}

//****************************************************************
//          Vole fills View
//****************************************************************
data "template_file" "volume_fills_query" {
  template = file("${path.module}/queries/volume_fills.sql")
  vars = {
    dataset = google_bigquery_dataset.serum.dataset_id
  }
}

resource "google_bigquery_table" "volume_fills" {
  dataset_id = google_bigquery_dataset.serum.dataset_id
  table_id   = "volume_fills"
  project                     = var.gcp_project

  view {
    query = data.template_file.volume_fills_query.rendered
    use_legacy_sql = false
  }

  depends_on = [google_bigquery_table.usd_priced_fills]
}