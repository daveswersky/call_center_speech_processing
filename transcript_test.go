package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	// [START imports]
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	// [END imports]
)

//Integration Test
func TestProcessTranscript(t *testing.T) {
	e := GCSEvent{}
	e.Bucket = os.Getenv("BUCKET_NAME")
	e.Name = os.Getenv("TEST_FILE")
	e.Metageneration = "1"
	Process_transcript(context.Background(), e)
}

func TestGetFileMetadata(t *testing.T) {
	ctx := context.Background()
	record := TranscriptRecord{}
	err := get_file_metadata(ctx, os.Getenv("BUCKET_NAME"), os.Getenv("TEST_FILE"), &record)
	if err != nil {
		t.Errorf("get_callid_from_audiofile: %v", err)
	}
	wants := "0987654321"
	if record.Callid != wants {
		t.Errorf("got %s, want %s", record.Callid, wants)
	}
	wants = "true"
	if record.Dlp != "true" {
		t.Errorf("got %s, want %s", record.Dlp, wants)
	}
}

func TestParseTranscript(t *testing.T) {
	jsonFile, err := ioutil.ReadFile("sample_transcript.json")
	if err != nil {
		t.Fatal(err)
	}
	record := TranscriptRecord{}
	result := speechpb.LongRunningRecognizeResponse{}
	err = json.Unmarshal(jsonFile, &result)
	if err != nil {
		t.Fatal(err)
	}
	err = parse_transcript(&result, &record)
	if err != nil {
		t.Errorf("parse_transcript: %v", err)
	}
	wants := []string{
		"Thank you for calling",
	}
	for _, want := range wants {
		if got := record.Transcript; !strings.Contains(got, want) {
			t.Errorf("got %s, want %s", got, want)
		}
	}
	wordCount := 455
	if len(record.Words) != wordCount {
		t.Errorf("got %d, want %d", len(record.Words), wordCount)
	}
	duration := 108.310
	if record.Duration != duration {
		t.Errorf("got %f, want %f", record.Duration, duration)
	}
}

func TestAudioTranscription(t *testing.T) {
	ctx := context.Background()
	err, resp := get_audio_transcript(ctx, fmt.Sprintf("gs://%s/%s", os.Getenv("BUCKET_NAME"), os.Getenv("TEST_FILE")))
	if err != nil {
		t.Errorf("get_audio_transcript: %v", err)
	}
	if resp.Results[0].Alternatives[0].Transcript != "Hi, I'd like to buy a Chromecast. I'm always wondering whether you could help me with that." {
		t.Errorf("got %s, want %s", resp.Results[0].Alternatives[0].Transcript, "Hello, how are you?")
	}
}

func TestRedactTranscript(t *testing.T) {
	ctx := context.Background()
	record := TranscriptRecord{}
	record.Transcript = "Hi my name is John Smith and my SSN is 123-45-6789. My home address is 555 Anystreet, Seattle WA 11010."
	err := redact_transcript(ctx, &record) ; if err != nil {
		t.Errorf("redact_transcript: %v", err)
	}
	wants := "Hi my name is ********** and my SSN is ***********. My home address is 555 Anystreet, ******* ** 11010."
	if wants != record.Transcript {
		t.Errorf("got %s, want %s", record.Transcript, wants)
	}
}

func TestGetSentiment(t *testing.T) {
	record := TranscriptRecord{}
	record.Transcript = "I am happy"
	record.Sentimentscore = 0.0
	ctx := context.Background()
	err := get_nlp_analysis(ctx, &record)
	if err != nil {
		t.Errorf("Error in get_sentiment_analysis: %v", err)
	}
	fmt.Println(record.Sentimentscore)
	if record.Sentimentscore != 0.900 {
		t.Errorf("got %s, want %s", fmt.Sprintf("%f",record.Sentimentscore), "0.0")
	}
}

func TestCommitBQ(t *testing.T) {
	ctx := context.Background()
	transcript := TranscriptRecord{}
	transcript.Date = time.Now()
	transcript.Fileid = "test"
	transcript.Transcript = "I am happy"
	transcript.Sentimentscore = 0.0
	transcript.Duration = 0.0
	transcript.Words = append(transcript.Words, struct {
		Word       string  `json:"word"`
		StartSecs  float64  `json:"startSecs"`
		EndSecs    float64  `json:"endSecs"`
		SpeakerTag int     `json:"speakertag"`
		Confidence float64 `json:"confidence"`
	}{
		Word:       "I",
		StartSecs:  0.0,
		EndSecs:    1.5,
		SpeakerTag: 1,
		Confidence: 0.9,
	})
	transcript.Entities = append(transcript.Entities, struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Sentiment float32 `json:"sentiment"`
	}{
		Name 	 : "I",
		Type 	 : "PERSON",
		Sentiment : 0.9,
	})
	transcript.Sentences = append(transcript.Sentences, struct {
		Sentence  string  `json:"sentence"`
		Sentiment float32 `json:"sentiment"`
		Magnitude float32 `json:"magnitude"`
	}{
		Sentence: "I am happy",
		Sentiment: 0.9,
		Magnitude: 0.9,
	})

	err := commit_transcript_record(ctx, &transcript)
	if err != nil {
		t.Errorf("Error in commit_bq: %v", err)
	}
}