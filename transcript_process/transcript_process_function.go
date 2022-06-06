package transcript_process_function

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"strconv"
	"strings"
	"cloud.google.com/go/functions/metadata"
	"github.com/golang/protobuf/proto"

	// [START imports]
	language "cloud.google.com/go/language/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	// [END imports]
)

// GCSEvent is the payload of a GCS event.
type GCSEvent struct {
	Kind                    string                 `json:"kind"`
	ID                      string                 `json:"id"`
	SelfLink                string                 `json:"selfLink"`
	Name                    string                 `json:"name"`
	Bucket                  string                 `json:"bucket"`
	Generation              string                 `json:"generation"`
	Metageneration          string                 `json:"metageneration"`
	ContentType             string                 `json:"contentType"`
	TimeCreated             time.Time              `json:"timeCreated"`
	Updated                 time.Time              `json:"updated"`
	TemporaryHold           bool                   `json:"temporaryHold"`
	EventBasedHold          bool                   `json:"eventBasedHold"`
	RetentionExpirationTime time.Time              `json:"retentionExpirationTime"`
	StorageClass            string                 `json:"storageClass"`
	TimeStorageClassUpdated time.Time              `json:"timeStorageClassUpdated"`
	Size                    string                 `json:"size"`
	MD5Hash                 string                 `json:"md5Hash"`
	MediaLink               string                 `json:"mediaLink"`
	ContentEncoding         string                 `json:"contentEncoding"`
	ContentDisposition      string                 `json:"contentDisposition"`
	CacheControl            string                 `json:"cacheControl"`
	Metadata                map[string]interface{} `json:"metadata"`
	CRC32C                  string                 `json:"crc32c"`
	ComponentCount          int                    `json:"componentCount"`
	Etag                    string                 `json:"etag"`
	CustomerEncryption      struct {
		EncryptionAlgorithm string `json:"encryptionAlgorithm"`
		KeySha256           string `json:"keySha256"`
	}
	KMSKeyName    string `json:"kmsKeyName"`
	ResourceState string `json:"resourceState"`
}

type TranscriptResult struct {
	Results []struct {
		Alternatives []struct {
			Confidence float64 `json:"confidence"`
			Transcript string  `json:"transcript"`
			Words      []struct {
				Confidence float64 `json:"confidence"`
				EndTime    string  `json:"endTime"`
				StartTime  string  `json:"startTime"`
				Word       string  `json:"word"`
			} `json:"words"`
		} `json:"alternatives"`
		ChannelTag    int    `json:"channelTag"`
		LanguageCode  string `json:"languageCode"`
		ResultEndTime string `json:"resultEndTime"`
	} `json:"results"`
}

type TranscriptRecord struct {
	fileid             string
	filename           string
	dlp                string
	callid             string
	date               time.Time
	year               int
	month              int
	day                int
	starttime          string
	duration           float64
	silencesecs        float64
	sentimentscore     float64
	magnitude          float64
	silencepercentage  int
	speakeronespeaking float64
	speakertwospeaking float64
	nlcategory         string
	transcript         string
	Words              []struct {
		Word       string  `json:"word"`
		StartSecs  string  `json:"startSecs"`
		EndSecs    string  `json:"endSecs"`
		SpeakerTag int     `json:"speakertag"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
	Entities []struct {
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Sentiment float64 `json:"sentiment"`
	} `json:"entities"`
	Sentences []struct {
		Sentence  string  `json:"sentence"`
		Sentiment float64 `json:"sentiment"`
		Magnitude float64 `json:"magnitude"`
	} `json:"sentences"`
}

//Triggered by Create/Finalize in the audio upload bucket
func process_transcript(ctx context.Context, event GCSEvent) error {
	//Read the metadata from the event
	meta, err := metadata.FromContext(ctx)
	record := TranscriptRecord{}

	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}
	fmt.Printf("Cloud Function triggered by change to: %v\n", meta.EventID)

	//Submit audio file to Google Speech API
	jsonText := ""

	//Build the transcript record
	parse_transcript_from_json(jsonText, &record)

	return nil
}

//Builds the transcript record from the transcript
func parse_transcript_from_json(rawJson string, record *TranscriptRecord) (error, string) {
	transcript := rawJson
	result := TranscriptResult{}
	
	err := json.Unmarshal([]byte(rawJson), &result)
	if err != nil {
		return err, transcript
	}
	for _, result := range result.Results {
		transcript += result.Alternatives[0].Transcript
	}

	//Build the transcript record
	record.transcript = transcript

	for _, result := range result.Results {
		for _, word := range result.Alternatives[0].Words {
			//Parse the Word start and end times
			start, err := strconv.ParseFloat(strings.ReplaceAll(word.StartTime, "s",""), 64)
				if err != nil {
					return err, transcript
				}
			end, err := strconv.ParseFloat(strings.ReplaceAll(word.EndTime, "s",""), 64)
				if err != nil {
					return err, transcript
				}
			//Incremenent the speaker durations
			if result.ChannelTag == 1 {
				record.speakeronespeaking += end - start
			} else {
				record.speakertwospeaking += end - start
			}
			record.Words = append(record.Words, struct {
				Word       string  `json:"word"`
				StartSecs  string  `json:"startSecs"`
				EndSecs    string  `json:"endSecs"`
				SpeakerTag int     `json:"speakertag"`
				Confidence float64 `json:"confidence"`
			}{
				Word:       word.Word,
				StartSecs:  word.StartTime,
				EndSecs:    word.EndTime,
				SpeakerTag: result.ChannelTag,
				Confidence: word.Confidence,
			})
		}
	}

	//Get duration by adding the first start time to the last end time
	duration, err := strconv.ParseFloat(strings.ReplaceAll(result.Results[len(result.Results)-1].ResultEndTime, "s",""), 64)
		if err != nil {	
			return err, transcript
		}
	record.duration = duration
	record.nlcategory = "N/A"
	ctx := context.Background()

	get_nlp_analysis(ctx, record)
	get_dlp_anaysis(record)
	return nil, transcript
}

//Get sentiment analysis from the Google Cloud Natural Language API
//AnalyzeSentiment and AnalayzeEntitySentiment 
func get_nlp_analysis(ctx context.Context, record *TranscriptRecord) error {
	//Get the sentiment analysis
	client, err := language.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.AnalyzeEntities(ctx, &languagepb.AnalyzeEntitiesRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: record.transcript,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
		EncodingType: languagepb.EncodingType_UTF8,
	})

	for _, entity := range resp.Entities {	
		fmt.Println(entity.Name, entity.Type, entity.Salience, entity.Mentions)
	}
	
	if err != nil {	
		return err
	}
	
	return nil
}

func get_dlp_anaysis(record *TranscriptRecord) error {

	return nil
}
