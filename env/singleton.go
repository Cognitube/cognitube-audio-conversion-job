package env

import (
	"os"
	"strconv"
	"sync"
)

type Variables struct {
	BlobConnectString   string
	TranscodingMaxRetry int
	DefaultAudioBitRate int
	AudioContainerName  string
	FFMPEGPath          string
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
		BlobConnectString:   os.Getenv("AZURE_BLOB_CONNECTION_STRING"),
		DefaultAudioBitRate: getEnvWithDefaultInt("DEFAULT_AUDIO_BITRATE", 128),
		TranscodingMaxRetry: 3,
		AudioContainerName:  os.Getenv("AUDIO_CONTAINER_NAME"),
		FFMPEGPath:          os.Getenv("FFMPEG_PATH"),
	}
}

func GetInstance() *Variables {
	if instance == nil {
		once.Do(loadValues)
	}
	return instance
}
