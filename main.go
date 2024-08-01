package main

import (
	"audio-extraction-job/env"
	"audio-extraction-job/result"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func parseBlobURL(blobURL string) (containerName, blobName string, err error) {
	// 解析 URL
	uri, err := url.Parse(blobURL)
	if err != nil {
		return "", "", fmt.Errorf("error parsing blob URL: %w", err)
	}

	// 获取路径部分
	path := uri.Path

	// 根据 '/' 分割路径
	parts := strings.Split(path, "/")

	// 检查分割后的部分是否足够多以包含容器名和 Blob 名称
	if len(parts) < 3 {
		return "", "", fmt.Errorf("path '%s' is too short, must include container and blob name", path)
	}

	// 容器名通常是第一个有效部分（跳过空字符串）
	containerName = parts[1]

	// Blob 名称是容器名之后的所有部分，合并为完整的 Blob 名称
	blobName = strings.Join(parts[2:], "/")

	return containerName, blobName, nil
}

func hasAudio(videoFile os.File) (bool, error) {
	cmd := exec.Command(env.GetInstance().FFMPEGPath, "-i", videoFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error while checking audio in video: %w", err)
	}
	return strings.Contains(string(output), "Audio"), nil
}

func convertToAudio(videoURL string) (string, error) {
	fmt.Println("Processing video from URL:", videoURL)

	containerName, blobName, err := parseBlobURL(videoURL)
	if err != nil {
		return "", fmt.Errorf("error parsing blob URL: %w", err)
	}
	client, _ := azblob.NewClientFromConnectionString(env.GetInstance().BlobConnectString, nil)

	log.Printf("Downloading blob: %s/%s", containerName, blobName)
	log.Println(env.GetInstance().BlobConnectString)
	containerClient := client.ServiceClient().NewContainerClient(containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	resp, err := blobClient.DownloadStream(context.Background(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to download blob: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read blob data: %w", err)
	}

	srcFile, err := os.CreateTemp("", "temp-")
	if err != nil {
		return "", fmt.Errorf("error while creating temp file: %w", err)
	}
	defer srcFile.Close()
	defer os.Remove(srcFile.Name())

	_, err = srcFile.Write(data)
	if err != nil {
		return "", fmt.Errorf("error while writing to temp file: %w", err)
	}

	videoHasAudio, err := hasAudio(*srcFile)
	if err != nil {
		return "", fmt.Errorf("error while checking audio in video: %w", err)
	}
	if !videoHasAudio {
		return "", fmt.Errorf("video does not have audio")
	}

	targetFile, err := os.CreateTemp("", "processed-*.mp4")
	if err != nil {
		return "", fmt.Errorf("error while creating temp file: %w", err)
	}
	defer targetFile.Close()
	defer os.Remove(targetFile.Name())

	// Convert video to audio using ffmpeg
	cmd := exec.Command(env.GetInstance().FFMPEGPath, "-i", srcFile.Name(), "-y", "-acodec", "libopus", "-b:a", strconv.Itoa(env.GetInstance().DefaultAudioBitRate)+"k", targetFile.Name())
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error while converting video to audio: %w", err)
	}

	_, err = client.UploadBuffer(context.TODO(), env.GetInstance().AudioContainerName, targetFile.Name(), data, nil)
	if err != nil {
		return "", fmt.Errorf("error while uploading audio to blob: %w", err)
	}

	blobURL := fmt.Sprintf("%s%s/%s", client.URL(), env.GetInstance().AudioContainerName, blobName)
	log.Printf("Audio uploaded to: %s", blobURL)
	return blobURL, nil
}

func initiateKeywordExtraction(videoID, audioURL string) {
	url := env.GetInstance().KeywordServiceURL + "/v1/transcription/create"
	data := map[string]string{
		"videoId":  videoID,
		"audioUrl": audioURL,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error while marshalling data: %v", err)
		return
	}

	_, err = http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error while sending request to keyword service: %v", err)
		return
	}
}

func main() {
	videoID := os.Args[1]
	videoURL := os.Args[2]
	kafkaTopic := os.Args[3]
	var audioURL string

	retry := -1
	for retry < env.GetInstance().TranscodingMaxRetry {
		audioLink, err := convertToAudio(videoURL)
		if err == nil {
			audioURL = audioLink
			break
		}

		if err.Error() == "video does not have audio" {
			log.Printf("Video does not have audio, skipping")

			msg, err := json.Marshal(result.Result{
				Success:       false,
				Error:         "video does not have audio",
				VideoID:       videoID,
				KeywordsURL:   "",
				TranscriptURL: "",
			})
			if err != nil {
				log.Printf("Error while marshalling result: %v", err)
				return
			}

			err = PublishProd(kafkaTopic, []byte(msg))
			if err != nil {
				log.Printf("Error while publishing message to Kafka: %v", err)
			}
			return
		}

		log.Printf("Error while converting video to audio in the %dth attempt: %v", retry+1, err)
		retry++
	}

	if audioURL == "" {
		initiateKeywordExtraction(videoID, audioURL)
	}
}
