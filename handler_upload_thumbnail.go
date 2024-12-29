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
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request body", err)
		return
	}

	mpFile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read FormFile", err)
		return
	}
	defer mpFile.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find video", err)
		return
	}
	randNum := make([]byte, 32)
	_, err = rand.Read(randNum)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not generator video ID", err)
	}

	var randomName = base64.RawURLEncoding.EncodeToString(randNum)
	parts := strings.Split(mediaType, "/")
	fileExtension := parts[1]
	fileName := fmt.Sprintf("%s.%s", randomName, fileExtension)
	pathURL := filepath.Join(cfg.assetsRoot, fileName)

	file, err := os.Create(pathURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating thumbnail on server", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, mpFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying thumbnail to server", err)
		return
	}

	newThumbnailURL := fmt.Sprintf("/assets/%s.%s", randomName, fileExtension)
	dbVideo.ThumbnailURL = &newThumbnailURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
