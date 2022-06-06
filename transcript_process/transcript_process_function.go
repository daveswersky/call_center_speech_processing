package transcript_process_function

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/functions/metadata"
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
	words              []string
	entities           []string
	sentences          []string
}

func process_transcript(ctx context.Context, event GCSEvent) error {
	//Read the metadata from the event
	meta, err := metadata.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}
	fmt.Printf("Cloud Function triggered by change to: %v\n", meta.EventID)

	//Read the json from the file

	return nil
}

func get_transcript_from_json(rawJson string) string {
	transcript := ""
	result := TranscriptResult{}
	json.Unmarshal([]byte(rawJson), &result)

	for _, result := range result.Results {
		transcript += result.Alternatives[0].Transcript
	}

	return transcript
}
