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

# Create a storage bucket for Uploaded Audio Files
resource "google_storage_bucket" "audio_uploads_bucket" {
  name = "${var.audio_uploads_bucket}-${random_id.bucket_id.hex}"
  location = var.bucket_location
}
# Grant privs to bucket for service account
resource "google_storage_bucket_iam_member" "audio_member" {
  bucket = google_storage_bucket.audio_uploads_bucket.name
  role = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}
# Create a storage bucket for Cloud Function Source files
resource "google_storage_bucket" "function_bucket" {
  name = "${var.function_bucket}-${random_id.bucket_id.hex}"
  location = var.bucket_location
}
# Create a BigQuery Dataset
resource "google_bigquery_dataset" "dataset" {
  dataset_id    = var.dataset_id
  friendly_name = "Transcripts Dataset"
  description   = "Call audio transcripts dataset"
  location      = "US"
}
# Create Cloud Function
resource "google_cloudfunctions2_function" "function" {
  provider    = google-beta
  name        = var.function_name
  location    = var.function_region  
  description = var.function_description
  build_config {
    runtime     = "go116"
    entry_point = "Process_transcript"
    source {
        storage_source {
            bucket = google_storage_bucket.function_bucket.name
            object = "function.zip"
        }
    }
  }
  service_config {
    max_instance_count = 3
    min_instance_count = 1
    available_memory  = 256
    service_account_email = google_service_account.service_account.email
    environment_variables = {
      "key" = "value"
    }
  }
  event_trigger {
    trigger_region = var.function_region
    trigger = google_storage_bucket.audio_uploads_bucket.name
    event_type = "google.storage.object.finalize"
  }
#   depends_on = [
#     data.archive_file.function_files,
#   ]
}