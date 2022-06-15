/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

# Configure the Google Cloud provider
provider "google" {
  project     = var.project_id
}

provider "google-beta" {
  project     = var.project_id
}

resource "random_id" "bucket_id" {
  byte_length = 8
}

# Enable APIs
resource "google_project_service" "project" {
  for_each = toset(var.service_ids)
  project = var.project_id
  service = each.value

  timeouts {
    create = "30m"
    update = "40m"
  }

  disable_dependent_services = true
}

resource "google_service_account" "service_account" {
  account_id   = "transcription-project-sa"
  display_name = "Project Service Account"
}

# Cloud Pub/Sub needs the role 
# roles/iam.serviceAccountTokenCreator granted to service 
# account service-859454537485@gcp-sa-pubsub.iam.gserviceaccount.com on this project to create identity tokens. You can change this later.

# This trigger needs the role 
# roles/pubsub.publisher granted to service 
# account service-859454537485@gs-project-accounts.iam.gserviceaccount.com 
# to receive events via Cloud Storage.


resource "google_project_iam_member" "log-binding" {
  project = var.project_id
  role = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}

resource "google_project_iam_member" "functions-binding" {
  project = var.project_id
  role = "roles/cloudfunctions.invoker"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}

resource "google_project_iam_member" "bigquery-binding" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}
resource "google_project_iam_member" "storage-binding" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}
resource "google_project_iam_member" "dlp-binding" {
  project = var.project_id
  role    = "roles/dlp.user"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}
# This trigger needs the role roles/eventarc.eventReceiver granted to service account
# saf-v2@appspot.gserviceaccount.com to receive events via Cloud Audit Logs.

