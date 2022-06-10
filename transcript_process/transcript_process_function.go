package transcript_process_function

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/functions/metadata"
	"github.com/kjk/betterguid"

	// [START imports]
	"cloud.google.com/go/bigquery"
	language "cloud.google.com/go/language/apiv1"
	speech "cloud.google.com/go/speech/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
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
	Fileid             string `json:"fileid"`
	Filename           string `json:"filename"`
	Dlp                string `json:"dlp"`
	Callid             string `json:"callid"`
	Date               time.Time `json:"date"`
	Year               int `json:"year"`
	Month              int `json:"month"`
	Day                int `json:"day"`
	Starttime          string `json:"starttime"`
	Duration           float64 `json:"duration"`
	Silencesecs        float64 `json:"silencesecs"`
	Sentimentscore     float32 `json:"sentimentscore"`
	Magnitude          float32 `json:"magnitude"`
	Silencepercentage  int `json:"silencepercentage"`
	Speakeronespeaking float64 `json:"speakeronespeaking"`
	Speakertwospeaking float64 `json:"speakertwospeaking"`
	Nlcategory         string `json:"nlcategory"`
	Transcript         string `json:"transcript"`
	Words              []struct {
		Word       string  `json:"word"`
		StartSecs  float64  `json:"startSecs"`
		EndSecs    float64  `json:"endSecs"`
		SpeakerTag int     `json:"speakertag"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
	Entities []struct {
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Sentiment float32 `json:"sentiment"`
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
	file := event
	record.Fileid = betterguid.New()
	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}
	fmt.Printf("Cloud Function triggered by change to: %v\n", meta.EventID)

	//Submit audio file to Google Speech API
	err, result := get_audio_transcript(ctx, fmt.Sprintf("gs://%s/%s", file.Bucket, file.Name))
	if err != nil {
		return err
	}

	//Build the transcript record
	parse_transcript(result, &record)
	//Get the sentiment analysis
	get_nlp_analysis(ctx, &record)
	//get_dlp_anaysis(record)
	
	return nil
}

func get_audio_transcript(ctx context.Context, gcsUri string) (error, *speechpb.LongRunningRecognizeResponse) {
	client, err := speech.NewClient(ctx)
	if err != nil {
		return err, nil
	}
	req :=  &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			SampleRateHertz:                     44100,
			LanguageCode:                        "en-US",
			Encoding:                            speechpb.RecognitionConfig_LINEAR16,
			AudioChannelCount:                   2,
			EnableSeparateRecognitionPerChannel: true,
			MaxAlternatives:                     0,
			EnableAutomaticPunctuation:          true,
			EnableWordTimeOffsets:               true,
			EnableWordConfidence:                true,
			UseEnhanced:                         true,
			Model:                               "phone_call",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsUri},
		},
	}

	op, err := client.LongRunningRecognize(ctx, req)

	if err != nil {
		return err, nil 
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return err, nil
	}
	return nil, resp
}

//Builds the transcript record from the transcript
func parse_transcript(transcript *speechpb.LongRunningRecognizeResponse, record *TranscriptRecord) error {
	transcriptText := ""

	for _, result := range transcript.Results {
		transcriptText += result.Alternatives[0].Transcript
	}

	//Build the transcript record
	record.Transcript = transcriptText

	for _, result := range transcript.Results {
		for _, word := range result.Alternatives[0].Words {
			//Parse the Word start and end times
			start, err := strconv.ParseFloat(word.StartTime.String(), 64)
			if err != nil {
				return err
			}
			end, err := strconv.ParseFloat(word.EndTime.String(), 64)
			if err != nil {
				return err
			}
			
			//Incremenent the speaker durations
			if result.ChannelTag == 1 {
				record.Speakeronespeaking += float64(end) - float64(start)
			} else {
				record.Speakertwospeaking += float64(end) - float64(start)
			}
			record.Words = append(record.Words, struct {
				Word       string  `json:"word"`
				StartSecs  float64  `json:"startSecs"`
				EndSecs    float64  `json:"endSecs"`
				SpeakerTag int     `json:"speakertag"`
				Confidence float64 `json:"confidence"`
			}{
				Word:       word.Word,
				StartSecs:  start,
				EndSecs:    end,
				SpeakerTag: int(result.ChannelTag),
				Confidence: float64(word.Confidence),
			})
		}
	}

	//Get duration by adding the first start time to the last end time
	duration := transcript.Results[len(transcript.Results)-1].ResultEndTime.Seconds //strconv.ParseFloat(strings.ReplaceAll(transcript.Results[len(transcript.Results)-1].ResultEndTime.String(), "s",""), 64)
		
	record.Duration = float64(duration)
	record.Silencesecs = float64(duration) - record.Speakeronespeaking - record.Speakertwospeaking
	record.Silencepercentage = int(record.Silencesecs / float64(duration) * 100)
	record.Nlcategory = "N/A"
	return nil
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

	r, err := client.AnalyzeSentiment(ctx, &languagepb.AnalyzeSentimentRequest{
		Document: &languagepb.Document{
				Source: &languagepb.Document_Content{
						Content: record.Transcript,
				},
				Type: languagepb.Document_PLAIN_TEXT,
		},
	})
	if err != nil {
		return err
	}

	record.Sentimentscore = r.DocumentSentiment.Score
	record.Magnitude = r.DocumentSentiment.Magnitude

	//Get the entity analysis
	entitySentiment, err := client.AnalyzeEntitySentiment(ctx, &languagepb.AnalyzeEntitySentimentRequest{
		Document: &languagepb.Document{
				Source: &languagepb.Document_Content{
						Content: record.Transcript,
				},
				Type: languagepb.Document_PLAIN_TEXT,
		},
	})
	if err != nil {
		return err
	}
	for _, entity := range entitySentiment.Entities {
		record.Entities = append(record.Entities, struct {
			Name       string  `json:"name"`
			Type       string  `json:"type"`
			Sentiment  float32  `json:"sentiment"`
		}{
			Name:       entity.Name,
			Type:       entity.Type.String(),
			Sentiment:  entity.Sentiment.Score,
		})
	}		
	return nil
}

func get_dlp_anaysis(record *TranscriptRecord) error {

	return nil
}

func commit_transcript_record(ctx context.Context, projectID, datasetID, tableID string, record *TranscriptRecord) error {
	//Commit the transcript record to the database
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()
	inserter := client.Dataset(datasetID).Table(tableID).Inserter()
	items := record
	if err := inserter.Put(ctx, items); err != nil {
		return err
	}

	return nil
}