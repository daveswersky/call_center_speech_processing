package transcript_process_function

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestGetTranscript(t *testing.T) {
	jsonFile, err := ioutil.ReadFile("sample.json")
	if err != nil {
		t.Fatal(err)
	}
	record := TranscriptRecord{}

	rawTranscript := string(jsonFile)
	err, transcript := parse_transcript_from_json(rawTranscript, &record)
	if err != nil {
		t.Errorf("get_transcript_from_json: %v", err)
	}

	wants := []string{
		"Thank you for calling",
		"Hey, Mika. I like to order flowers from your store.",
	}

	for _, want := range wants {
		if got := transcript; !strings.Contains(got, want) {
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
