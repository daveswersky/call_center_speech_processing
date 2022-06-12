package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	// [START imports]
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	// [END imports]
)


func TestProcessTranscript(t *testing.T) {
	e := GCSEvent{}
	e.Bucket = ""
	e.Name = "test.wav"
	e.Metageneration = "1"

	Process_transcript(context.Background(), e)
}

func TestGetCallid(t *testing.T) {
	ctx := context.Background()
	callid, err := get_callid_from_audiofile(ctx,"safv2_audio_upload", "sample_order.wav")
	if err != nil {
		t.Errorf("get_callid_from_audiofile: %v", err)
	}
	wants := "123456"
	if callid != wants {
		t.Errorf("got %s, want %s", callid, wants)
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

	//fmt.Println(record.transcript)
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

	duration := 111.1
	if record.Duration != duration {
		t.Errorf("got %f, want %f", record.Duration, duration)
	}
}

func TestAudioTranscription(t *testing.T) {
	ctx := context.Background()
	err, resp := get_audio_transcript(ctx, "gs://saf-audio-6bc68142dfd12f49/commercial_stereo.wav")
	if err != nil {
		t.Errorf("get_audio_transcript: %v", err)
	}

	if resp.Results[0].Alternatives[0].Transcript != "Hi, I'd like to buy a Chromecast. I'm always wondering whether you could help me with that." {
		t.Errorf("got %s, want %s", resp.Results[0].Alternatives[0].Transcript, "Hello, how are you?")
	}
}

func TestGetSentiment(t *testing.T) {
	record := TranscriptRecord{}
	record.Transcript = "I am happy"
	record.Sentimentscore = 0.0
	ctx := context.Background()
	err := get_nlp_analysis(ctx, &record)
	if err != nil {
		t.Errorf("get_sentiment_analysis: %v", err)
	}
	fmt.Println(record.Sentimentscore)
	if record.Sentimentscore != 0.900 {
		t.Errorf("got %s, want %s", fmt.Sprintf("%f",record.Sentimentscore), "0.0")
	}
}

func TestCommitBQ(t *testing.T) {
	os.Setenv("GOOGLE_CLOUD_PROJECT", "saf-v2")
	os.Setenv("GOOGLE_DATASET_ID", "saf")
	os.Setenv("GOOGLE_TABLE_ID", "transcripts")
	ctx := context.Background()
	transcript := TranscriptRecord{}
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
		t.Errorf("commit_bq: %v", err)
	}
}