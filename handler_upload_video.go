package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// We're limiting the max upload size
	const maxUpload = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by", userID)

	vidMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch vid data from the database", err)
		return
	}
	if vidMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Operation unauthorized", err)
		return
	}

	const maxMemory = 1 << 30
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse form file", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	filetype, _, err := mime.ParseMediaType(header.Header["Content-Type"][0])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media information", err)
		return
	}
	if filetype != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong file type. Only mp4 files allowed", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp file on server", err)
		return
	}

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write to temp file on server", err)
		return
	}

	tmpFile.Close()

	// Converting the video for a faster start in browser
	convFPath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing the video", err)
		return
	}

	// The tmp file is not needed
	os.Remove(tmpFile.Name())

	processedFile, err := os.Open(convFPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing the video", err)
		return
	}

	defer processedFile.Close()
	defer os.Remove(processedFile.Name())

	aspectRatio, err := getVideoAspectRatio(processedFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to check the aspect ratio of video", err)
		return
	}

	var aspectPrefix string
	if aspectRatio == "16:9" || aspectRatio == "16:10" {
		aspectPrefix = "landscape"
	} else if aspectRatio == "9:16" || aspectRatio == "10:16" {
		aspectPrefix = "portrait"
	} else {
		aspectPrefix = "other"
	}

	ext := strings.TrimPrefix(filetype, "video/")
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	randomName := base64.RawURLEncoding.EncodeToString(randomBytes)
	fname := fmt.Sprint(aspectPrefix, "/", string(randomName), ".", ext)

	poParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fname,
		Body:        processedFile,
		ContentType: &filetype,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &poParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload the file to the file server", err)
		return
	}

	//vidURL := fmt.Sprint("https://", cfg.s3Bucket, ".s3.", cfg.s3Region, ".amazonaws.com/", fname)
	// We generate a url to the video, and also add in the bucket name and key to it
	//vidURL := fmt.Sprint(cfg.s3Bucket, ",", fname)
	// Now we use CloudFront to distribute the content
	vidURL := fmt.Sprint("https://", cfg.s3CfDistribution, "/", fname)
	vidMetadata.VideoURL = &vidURL
	if err = cfg.db.UpdateVideo(vidMetadata); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload vid", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vidMetadata)
}
