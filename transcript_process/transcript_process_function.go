package transcript_process_function

import (
	"context"

	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/functions/metadata"

	// [START imports]
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
	sentimentscore     float32
	magnitude          float32
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
	result := TranscriptResult{}

	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}
	fmt.Printf("Cloud Function triggered by change to: %v\n", meta.EventID)

	//Submit audio file to Google Speech API
	//result = nil

	//Build the transcript record
	parse_transcript(&result, &record)
	//Get the sentiment analysis
	get_nlp_analysis(ctx, &record)
	//get_dlp_anaysis(record)
	
	return nil
}

func get_audio_transcript(client *speech.Client, ctx context.Context, gcsUri string) (error, *speechpb.LongRunningRecognizeResponse) {
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
func parse_transcript(transcript *TranscriptResult, record *TranscriptRecord) error {
	transcriptText := ""
	// result := TranscriptResult{}
	
	// err := json.Unmarshal([]byte(rawJson), &result)
	// if err != nil {
	// 	return err, transcript
	// }
	// for _, result := range result.Results {
	// 	transcript += result.Alternatives[0].Transcript
	// }

	for _, result := range transcript.Results {
		transcriptText += result.Alternatives[0].Transcript
	}

	//Build the transcript record
	record.transcript = transcriptText

	for _, result := range transcript.Results {
		for _, word := range result.Alternatives[0].Words {
			//Parse the Word start and end times
			start, err := strconv.ParseFloat(strings.ReplaceAll(word.StartTime, "s",""), 64)
				if err != nil {
					return err
				}
			end, err := strconv.ParseFloat(strings.ReplaceAll(word.EndTime, "s",""), 64)
				if err != nil {
					return err
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
	duration, err := strconv.ParseFloat(strings.ReplaceAll(transcript.Results[len(transcript.Results)-1].ResultEndTime, "s",""), 64)
		if err != nil {	
			return err
		}
	record.duration = duration
	record.silencesecs = duration - record.speakeronespeaking - record.speakertwospeaking
	record.silencepercentage = int(record.silencesecs / duration * 100)
	record.nlcategory = "N/A"
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
						Content: record.transcript,
				},
				Type: languagepb.Document_PLAIN_TEXT,
		},
	})
	if err != nil {
		return err
	}

	record.sentimentscore = r.DocumentSentiment.Score
	record.magnitude = r.DocumentSentiment.Magnitude
	
	return nil
}

func get_dlp_anaysis(record *TranscriptRecord) error {

	return nil
}
