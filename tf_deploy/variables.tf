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

variable "project_id" {
  type        = string
  description = "GCP project name"
}

variable "service_account_email" {
  type        = string
  description = "Service account email"
}

variable "audio_uploads_bucket" {
  type        = string
  description = "Audio uploads bucket name"
  default     = "audio-upload"
}

variable "dataset_id" {
  type        = string
  description = "BigQuery dataset name"
  default     = "call_transcripts"
}

variable "table_id" {
  type        = string
  description = "BigQuery table name"
  default     = "transcripts"
}

variable "function_name" {
  type        = string
  description = "Cloud Function Name"
  default     = "call-audio-transcription"
}

variable "function_description" {
  type        = string
  description = "Cloud Function description"
  default     = "Call Audio Transcription Function"  
}

variable "function_region" {
  type        = string
  description = "Cloud Function Region"
  default     = "us-central1"
}

variable "function_bucket" {
  type        = string
  description = "Cloud Function source files"
  default     = "call-audio-function-source"
}

variable "bucket_location" {
  type        = string
  default = "us-central1"
  description = "Location"
}

variable "service_ids" {
    type = list(string)
    default = ["speech.googleapis.com",
      "pubsub.googleapis.com",
      "artifactregistry.googleapis.com",
      "cloudrunadmin.googleapis.com",
      "eventarc.googleapis.com",
      "build.googleapis.com",
      "cloudbuild.googleapis.com",
      "language.googleapis.com",
      "dlp.googleapis.com",
      "bigquery.googleapis.com",
      "cloudfunctions.googleapis.com",
      "monitoring.googleapis.com"]
}