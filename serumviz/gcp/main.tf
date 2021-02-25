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