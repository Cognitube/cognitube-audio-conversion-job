package env

import (
	"os"
	"strconv"
	"sync"
)

type Variables struct {
	BlobConnectString             string
	TranscodingMaxRetry           int
	DefaultAudioBitRate           int
	AudioContainerName            string
	FFMPEGPath                    string
	KafkaEventHubConnectionString string
	KafkaEventHubNamespace        string
	KeywordServiceURL             string
}

var instance *Variables
var once sync.Once

func getEnvWithDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func loadValues() {
	instance = &Variables{
		BlobConnectString:             os.Getenv("AZURE_BLOB_CONNECTION_STRING"),
		DefaultAudioBitRate:           getEnvWithDefaultInt("DEFAULT_AUDIO_BITRATE", 128),
		TranscodingMaxRetry:           3,
		AudioContainerName:            os.Getenv("AUDIO_CONTAINER_NAME"),
		FFMPEGPath:                    os.Getenv("FFMPEG_PATH"),
		KafkaEventHubConnectionString: os.Getenv("KAFKA_EVENTHUB_CONNECTION_STRING"),
		KafkaEventHubNamespace:        os.Getenv("KAFKA_EVENTHUB_NAMESPACE"),
		KeywordServiceURL:             os.Getenv("KEYWORD_SERVICE_URL"),
	}
}

func GetInstance() *Variables {
	if instance == nil {
		once.Do(loadValues)
	}
	return instance
}
