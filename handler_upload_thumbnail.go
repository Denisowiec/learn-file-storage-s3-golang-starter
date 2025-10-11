package main

import (
	"fmt"
	"io"
	"net/http"

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

	filetype := header.Header["Content-Type"][0]
	filecontent, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading the file", err)
		return
	}
	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch video data from the database", err)
	}
	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized operation", err)
		return
	}

	tb := thumbnail{
		data:      filecontent,
		mediaType: filetype,
	}
	videoThumbnails[videoID] = tb
	tbURL := fmt.Sprintf("http://localhost:8091/api/thumbnails/%s", videoID.String())
	metadata.ThumbnailURL = &tbURL

	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update the video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}
