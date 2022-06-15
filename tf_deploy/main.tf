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
# Grant privs to service account
resource "google_storage_bucket_iam_member" "audio_member" {
  bucket = google_storage_bucket.audio_uploads_bucket.name
  role = "roles/storage.admin"
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
# Create a BigQuery Table
resource "google_bigquery_table" "default" {
  dataset_id = google_bigquery_dataset.dataset.dataset_id
  table_id   = "bar"

  time_partitioning {
    type = "DAY"
  }

  labels = {
    env = "default"
  }

  schema = <<EOF
[
    {
        "mode": "NULLABLE", 
        "name": "fileid", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "filename", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "dlp", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "callid", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "date", 
        "type": "TIMESTAMP"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "year", 
        "type": "INTEGER"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "month", 
        "type": "INTEGER"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "day", 
        "type": "INTEGER"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "starttime", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "duration", 
        "type": "FLOAT"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "silencesecs", 
        "type": "FLOAT"
    },
    {
        "mode": "NULLABLE", 
        "name": "sentimentscore", 
        "type": "FLOAT"
    },
    {
        "mode": "NULLABLE", 
        "name": "magnitude", 
        "type": "FLOAT"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "silencepercentage", 
        "type": "INTEGER"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "speakeronespeaking", 
        "type": "FLOAT"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "speakertwospeaking", 
        "type": "FLOAT"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "nlcategory", 
        "type": "STRING"
    }, 
    {
        "mode": "NULLABLE", 
        "name": "transcript", 
        "type": "STRING"
    }, 
    {
        "fields": [
        {
            "mode": "NULLABLE", 
            "name": "name", 
            "type": "STRING"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "type", 
            "type": "STRING"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "sentiment", 
            "type": "FLOAT"
        }
        ], 
        "mode": "REPEATED", 
        "name": "entities", 
        "type": "RECORD"
    }, 
    {
        "fields": [
        {
            "mode": "NULLABLE", 
            "name": "word", 
            "type": "STRING"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "startSecs", 
            "type": "FLOAT"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "endSecs", 
            "type": "FLOAT"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "speakertag", 
            "type": "INTEGER"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "confidence", 
            "type": "FLOAT"
        }
        ], 
        "mode": "REPEATED", 
        "name": "words", 
        "type": "RECORD"
    }, 
    {
        "fields": [
        {
            "mode": "NULLABLE", 
            "name": "sentence", 
            "type": "STRING"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "sentiment", 
            "type": "FLOAT"
        }, 
        {
            "mode": "NULLABLE", 
            "name": "magnitude", 
            "type": "FLOAT"
        }
        ], 
        "mode": "REPEATED", 
        "name": "sentences", 
        "type": "RECORD"
        }
]
EOF
}
# Create function zipfie
data "archive_file" "function_files" {
    type        = "zip"
    output_path = "../function.zip"
    source {
        content = "../go.mod"
        filename = "go.mod"
    }
    source {
        content = "../go.sum"
        filename = "go.sum"
    }
    source {
        content = "../transcript_process.go"
        filename = "transcript_process.go"
    }
}
# Upload function source files to storage bucket
resource "google_storage_bucket_object" "archive" {
  name   = "function.zip"
  bucket = google_storage_bucket.function_bucket.name
  source = "../function.zip"
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
        "GOOGLE_CLOUD_PROJECT" = var.project_id
        "GOOGLE_DATASET_ID" = var.dataset_id
        "GOOGLE_TABLE_ID" = var.table_id
    }
  }
  event_trigger {
    trigger_region = var.function_region
    trigger = google_storage_bucket.audio_uploads_bucket.name
    event_type = "google.storage.object.finalize"
  }
  depends_on = [
    data.archive_file.function_files,
  ]
}