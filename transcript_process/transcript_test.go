package transcript_process_function

import (
	"os"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"encoding/json"
)

func TestGetTranscript(t *testing.T) {
	jsonFile, err := ioutil.ReadFile("sample.json")
	if err != nil {
		t.Fatal(err)
	}
	record := TranscriptRecord{}
	result := TranscriptResult{}
	err = json.Unmarshal([]byte(jsonFile), &result)
	if err != nil {
		t.Errorf("get_transcript_from_json: %v", err)
	}

	err = parse_transcript(&result, &record)
	if err != nil {
		t.Errorf("get_transcript_from_json: %v", err)
	}

	fmt.Println(record.transcript)
	wants := []string{
		"Thank you for calling",
		"Hey, Mika. I like to order flowers from your store.",
	}

	for _, want := range wants {
		if got := record.transcript; !strings.Contains(got, want) {
			t.Errorf("got %s, want %s", got, want)
		}
	}

	wordCount := 559
	if len(record.Words) != wordCount {
		t.Errorf("got %d, want %d", len(record.Words), wordCount)
	}

	duration := 208.930
	if record.duration != duration {
		t.Errorf("got %f, want %f", record.duration, duration)
	}
}

func TestAudioTranscription(t *testing.T) {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./saf-15-23693e6f9d80.json")
	ctx := context.Background()
	err, resp := get_audio_transcript(ctx, "gs://saf-audio-6bc68142dfd12f49/commercial_stereo.wav")
	if err != nil {
		t.Errorf("get_audio_transcript: %v", err)
	}

	if resp.Results[0].Alternatives[0].Transcript != "Hello, how are you?" {
		t.Errorf("got %s, want %s", resp.Results[0].Alternatives[0].Transcript, "Hello, how are you?")
	}
}

func TestGetSentiment(t *testing.T) {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./saf-15-23693e6f9d80.json")
	record := TranscriptRecord{}
	record.transcript = "I am happy"
	record.sentimentscore = 0.0
	ctx := context.Background()
	err := get_nlp_analysis(ctx, &record)
	if err != nil {
		t.Errorf("get_sentiment_analysis: %v", err)
	}
	fmt.Println(record.sentimentscore)
	if record.sentimentscore != 0.900 {
		t.Errorf("got %s, want %s", fmt.Sprintf("%f",record.sentimentscore), "0.0")
	}
}