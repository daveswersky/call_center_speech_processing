package function

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kjk/betterguid"

	// [START imports]
	"cloud.google.com/go/bigquery"
	dlp "cloud.google.com/go/dlp/apiv2"
	language "cloud.google.com/go/language/apiv1"
	"cloud.google.com/go/logging"
	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	dlppb "google.golang.org/genproto/googleapis/privacy/dlp/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	// [END imports]
)


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
		Sentiment float32 `json:"sentiment"`
		Magnitude float32 `json:"magnitude"`
	} `json:"sentences"`
} 

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

//Triggered by Create/Finalize in the audio upload bucket
func Process_transcript(ctx context.Context, e GCSEvent) error {
	record := TranscriptRecord{}
	err := confirm_env_vars() ; if err != nil {
		log.Fatalf("Missing environment variables: %v", err)
	}
	logger, err := logging.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Failed to create logging client: %v", err)
	}
	//Read the metadata from the file
	err = get_file_metadata(ctx, e.Bucket, e.Name, &record) 
	if err != nil { 
		log.Fatalf("Failed to get metadata from audio file: %v", err) 
	}
	file := e
	record.Date = time.Now()
	record.Fileid = betterguid.New()
	record.Filename = fmt.Sprintf("%s/%s", file.Bucket, file.Name)
	writeEntry(logger, logging.Info, "Processing audio for callid: " + record.Callid + " | eventId: " + e.ID)
	//Submit audio file to Google Speech API
	err, result := get_audio_transcript(ctx, fmt.Sprintf("gs://%s/%s", file.Bucket, file.Name))
	if err != nil {
		writeEntry(logger, logging.Critical, fmt.Sprintf("CALLID: %s | Failed to get transcript from audio file: %v", record.Callid, err))
		//return err
	}
	//Build the transcript record
	err = parse_transcript(result, &record) ; if err != nil {
		writeEntry(logger, logging.Critical, fmt.Sprintf("CALLID: %s | Failed to parse transcript from audio file: %v", record.Callid, err))
		//return err
	}
	//Use DLP to redact sensitive data
	if record.Dlp == "true" {
		err = redact_transcript(ctx, &record) ; if err != nil {
			writeEntry(logger, logging.Critical, fmt.Sprintf("CALLID: %s | Failed to get DLP analysis from audio file: %v", record.Callid, err))
			//return err
		}
	}
	//Get the sentiment analysis
	err = get_nlp_analysis(ctx, &record) ; if err != nil {
		writeEntry(logger, logging.Critical, fmt.Sprintf("CALLID: %s | Failed to get sentiment analysis from audio file: %v", record.Callid, err))
		//return err
	}
	//Commit BQ record
	err = commit_transcript_record(ctx, &record) ; if err != nil {
		writeEntry(logger, logging.Critical, fmt.Sprintf("CALLID: %s | Failed to commit transcript record to BigQuery: %v", record.Callid, err))
		//return err
	}
	writeEntry(logger, logging.Info, "Completed processing transcript for callid: " + record.Callid)
	return nil
}

func writeEntry(client *logging.Client, info logging.Severity, msg string) {
	logger := client.Logger("call-audio-processor")
	defer logger.Flush()
	log := logger.StandardLogger(info)
	log.Printf(msg)
}

func confirm_env_vars() error {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	datasetID := os.Getenv("GOOGLE_DATASET_ID")
	if datasetID == "" {
		return fmt.Errorf("GOOGLE_DATASET_ID environment variable not set")
	}
	tableID := os.Getenv("GOOGLE_TABLE_ID")
	if tableID == "" {
		return fmt.Errorf("GOOGLE_TABLE_ID environment variable not set")
	}
	return nil
}

func get_audio_samplerate(ctx context.Context, bucket, file string) (int32, error) {
	client, err := storage.NewClient(ctx) ; if err != nil {
		return 0, err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	rc, err := client.Bucket(bucket).Object(file).NewRangeReader(ctx, 0, 44)
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	header, err := ioutil.ReadAll(rc) ; if err != nil {
		return 0, err
	}
	sampleRate := bits32ToInt(header[24:28])
	return sampleRate, nil
}

// turn a 32-bit byte array into an int
func bits32ToInt(b []byte) int32 {
	if len(b) != 4 {
		panic("Expected size 4!")
	}
	var payload uint32
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.LittleEndian, &payload)
	if err != nil {
		panic(err)
	}
	return int32(payload) // easier to work with ints
}

func get_file_metadata(ctx context.Context, bucket, filename string, record *TranscriptRecord) error {
	//Get the metadata from the audio file
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	file := client.Bucket(bucket).Object(filename)
	attrs, err := file.Attrs(ctx)
	if err != nil {
		return err
	}
	record.Callid = attrs.Metadata["callid"]
	record.Dlp = attrs.Metadata["dlp"]
	return nil
}

func get_audio_transcript(ctx context.Context, gcsUri string) (error, *speechpb.LongRunningRecognizeResponse) {
	client, err := speech.NewClient(ctx)
	file := strings.Split(gcsUri, "/")
	bucketName := file[2]
	fileName := file[3]
	sampleRate, err := get_audio_samplerate(ctx, bucketName, fileName) ; 
	if err != nil {
		return err, nil
	}
	req :=  &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			SampleRateHertz:                     sampleRate,
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

func get_seconds_from_duration(duration *durationpb.Duration) float64 {
	return float64(duration.Seconds) + float64(duration.Nanos) / 1e9
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
			start := get_seconds_from_duration(word.StartTime)
			end := get_seconds_from_duration(word.EndTime)
			//Incremenent the speaker durations
			if result.ChannelTag == 1 {
				record.Speakeronespeaking += end - start
			} else {
				record.Speakertwospeaking += end - start
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
	duration := get_seconds_from_duration(transcript.Results[len(transcript.Results)-1].ResultEndTime)
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
	for _, entity := range r.Sentences {
		record.Sentences = append(record.Sentences, struct {
			Sentence string `json:"sentence"`
			Sentiment float32 `json:"sentiment"`
			Magnitude float32 `json:"magnitude"`
		}{
			Sentence: entity.Text.Content,
			Sentiment: entity.Sentiment.Score,
			Magnitude: entity.Sentiment.Magnitude,
		})
	}
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

func redact_transcript(ctx context.Context, record *TranscriptRecord) error {
	//Get the DLP analysis
	client, err := dlp.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	var infoTypes []*dlppb.InfoType
	req := &dlppb.DeidentifyContentRequest{
		Parent: "projects/" + os.Getenv("GOOGLE_CLOUD_PROJECT"),
		InspectConfig: &dlppb.InspectConfig{
			InfoTypes: infoTypes,
		},
		DeidentifyConfig: &dlppb.DeidentifyConfig{
			Transformation: &dlppb.DeidentifyConfig_InfoTypeTransformations{
				InfoTypeTransformations: &dlppb.InfoTypeTransformations{
					Transformations: []*dlppb.InfoTypeTransformations_InfoTypeTransformation{
						{
							InfoTypes: []*dlppb.InfoType{}, // Match all info types.
							PrimitiveTransformation: &dlppb.PrimitiveTransformation{
								Transformation: &dlppb.PrimitiveTransformation_CharacterMaskConfig{
									CharacterMaskConfig: &dlppb.CharacterMaskConfig{
										MaskingCharacter: "*",
									},
								},
							},
						},
					},
				},
			},
		},
		Item : &dlppb.ContentItem{
			DataItem: &dlppb.ContentItem_Value{
				Value: record.Transcript,
			},
		},
	}
	//Redact the combined transcript
	dlpResponse, err := client.DeidentifyContent(ctx, req) ; if err != nil {
		return err
	}
	record.Transcript = dlpResponse.GetItem().GetValue()
	//Redact the individual sentences
	for i, sentence := range record.Sentences {
		req.Item = &dlppb.ContentItem{
			DataItem: &dlppb.ContentItem_Value{
				Value: sentence.Sentence,
			},
		}
		dlpResponse, err = client.DeidentifyContent(ctx, req) ; if err != nil {
			return err
		}
		record.Sentences[i].Sentence = dlpResponse.GetItem().GetValue()
	}
	//Redact the individual words
	for i, word := range record.Words {
		req.Item = &dlppb.ContentItem{
			DataItem: &dlppb.ContentItem_Value{
				Value: word.Word,
			},
		}
		dlpResponse, err = client.DeidentifyContent(ctx, req) ; if err != nil {
			return err
		}
	    record.Words[i].Word = dlpResponse.GetItem().GetValue()
	}
	//Redact the individual entities
	for i, entity := range record.Entities {
		req.Item = &dlppb.ContentItem{
			DataItem: &dlppb.ContentItem_Value{
				Value: entity.Name,
			},
		}
		dlpResponse, err = client.DeidentifyContent(ctx, req) ; if err != nil {
			return err
		}
		record.Entities[i].Name = dlpResponse.GetItem().GetValue()
	}
	return nil
}

func commit_transcript_record(ctx context.Context, record *TranscriptRecord) error {
	//Get environment variables
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}
	datasetID := os.Getenv("GOOGLE_DATASET_ID")
	if datasetID == "" {
		return fmt.Errorf("GOOGLE_DATASET_ID environment variable not set")
	}
	tableID := os.Getenv("GOOGLE_TABLE_ID")
	if tableID == "" {
		return fmt.Errorf("GOOGLE_TABLE_ID environment variable not set")
	}
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