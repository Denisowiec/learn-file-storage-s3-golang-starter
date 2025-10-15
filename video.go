package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func getVideoAspectRatio(filepath string) (string, error) {
	type vidAspectJSON struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)

	output := bytes.Buffer{}
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	vidAspect := vidAspectJSON{}

	err = json.Unmarshal(output.Bytes(), &vidAspect)
	if err != nil {
		return "", err
	}

	w := vidAspect.Streams[0].Width
	h := vidAspect.Streams[0].Height

	if w/16 == h/9 {
		return "16:9", nil
	} else if w/9 == h/16 {
		return "9:16", nil
	} else if w/16 == h/10 {
		return "16:10", nil
	} else if w/10 == h/16 {
		return "10:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := fmt.Sprintf("%s.processing", filePath)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func generatePresiugnedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	params := s3.GetObjectInput{Bucket: &bucket, Key: &key}
	req, err := presignedClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	printVidData(video)
	splitUrl := strings.Split(*video.VideoURL, ",")
	if len(splitUrl) != 2 {
		// return video, nil
		return database.Video{}, fmt.Errorf("error retrieving bucket name and key")
	}
	bucket := splitUrl[0]
	key := splitUrl[1]
	newUrl, err := generatePresiugnedURL(cfg.s3Client, bucket, key, time.Minute)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &newUrl

	return video, nil
}

func printVidData(vid database.Video) {
	fmt.Println("id: ", vid.ID)
	fmt.Println("CreatedAt: ", vid.CreatedAt)
	fmt.Println("UpdatedAt: ", vid.UpdatedAt)
	fmt.Println("Title: ", vid.Title)
	fmt.Println("Description: ", vid.Description)
	fmt.Println("ThumbnailUrl: ", *vid.ThumbnailURL)
	fmt.Println("VideoURL: ", *vid.VideoURL)
	fmt.Println("UserID: ", vid.UserID)
}
