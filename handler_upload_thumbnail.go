package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
	}
	defer file.Close()

	filetype, _, err := mime.ParseMediaType(header.Header["Content-Type"][0])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media information", err)
		return
	}
	if filetype != "image/png" && filetype != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "Wrong file type. Only png and jpeg files allowed", err)
		return
	}

	// We save the thumbnail file on disk
	ext := strings.TrimPrefix(filetype, "image/")
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	randomName := base64.RawURLEncoding.EncodeToString(randomBytes)

	fname := fmt.Sprint(string(randomName), ".", ext)
	fpath := filepath.Join(cfg.assetsRoot, fname)
	tbfile, err := os.Create(fpath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save the file", err)
		return
	}
	defer tbfile.Close()

	if _, err = io.Copy(tbfile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save the thumbnail file", err)
	}

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch video data from the database", err)
	}
	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized operation", err)
		return
	}

	tbURL := fmt.Sprint("http://localhost:8091/assets/", fname)
	metadata.ThumbnailURL = &tbURL

	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update the video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}
