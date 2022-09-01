# Call Center Speech Processing

This repository contains a solution designed to process call center audio files. WAV-formatted audio is uploaded to a GCS bucket, then the solution will:

* Transcribe the audio
* Perform Sentiment analysis on the text, words, and each sentence
* Optionally redact PII from the transcribed text
* Commit the complete analysis record to BigQuery


 The solution combines the following Google Cloud services:
* Cloud Function (2nd Generation)
* Speech to Text
* Natural Language Processing
* Data Loss Prevention
* BigQuery


