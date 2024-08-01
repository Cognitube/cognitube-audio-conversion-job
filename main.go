package main

import (
	"audio-extraction-job/env"
	"context"
	"fmt"
	"io"
	"log"
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

func main() {
	videoURL := os.Args[1]
	fmt.Println("Processing video from URL:", videoURL)

	containerName, blobName, err := parseBlobURL(videoURL)
	if err != nil {
		panic(err)
	}
	client, _ := azblob.NewClientFromConnectionString(env.GetInstance().BlobConnectString, nil)

	log.Printf("Downloading blob: %s/%s", containerName, blobName)
	log.Println(env.GetInstance().BlobConnectString)
	containerClient := client.ServiceClient().NewContainerClient(containerName)
	blobClient := containerClient.NewBlobClient(blobName)

	resp, err := blobClient.DownloadStream(context.Background(), nil)
	if err != nil {
		log.Panicf("failed to download blob: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("failed to read blob data: %v", err)
	}

	srcFile, err := os.CreateTemp("", "temp-")
	if err != nil {
		log.Panicf("Error while creating temp file: %v", err)
	}
	defer srcFile.Close()
	defer os.Remove(srcFile.Name())

	_, err = srcFile.Write(data)
	if err != nil {
		log.Panicf("Error while writing to temp file: %v", err)
	}

	targetFile, err := os.CreateTemp("", "processed-*.mp4")
	if err != nil {
		log.Panicf("Error while creating temp file: %v", err)
	}
	defer targetFile.Close()
	defer os.Remove(targetFile.Name())

	// Convert video to audio using ffmpeg
	cmd := exec.Command(env.GetInstance().FFMPEGPath, "-i", srcFile.Name(), "-y", "-acodec", "libopus", "-b:a", strconv.Itoa(env.GetInstance().DefaultAudioBitRate)+"k", targetFile.Name())
	if err := cmd.Run(); err != nil {
		log.Panicf("Error while converting video to audio: %v", err)
	}

	_, err = client.UploadBuffer(context.TODO(), env.GetInstance().AudioContainerName, targetFile.Name(), data, nil)
	if err != nil {
		log.Panicf("Error while uploading audio to blob: %v", err)
	}

	blobURL := fmt.Sprintf("%s%s/%s", client.URL(), env.GetInstance().AudioContainerName, blobName)
	log.Printf("Audio uploaded to: %s", blobURL)
}
