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

	rawTranscript := string(jsonFile)
	transcript := get_transcript_from_json(rawTranscript)

	wants := []string{
		"Thank you for calling",
		"Hey, Mika. I like to order flowers from your store.",
	}

	for _, want := range wants {
		if got := transcript; !strings.Contains(got, want) {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}
