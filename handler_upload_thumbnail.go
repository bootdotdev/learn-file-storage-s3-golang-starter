package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not parse multiform", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "the video ID does not exists", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorised", err)
		return
	}

	mediaExt := splitMediaType(mediaType)

	fileName := fmt.Sprintf("/%s.%s", videoID, mediaExt)
	filePath := filepath.Join(cfg.assetsRoot, fileName)

	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create a new file", err)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not copy the file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID, mediaExt)

	err = cfg.db.UpdateVideo(database.Video{
		ID:                videoID,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      &thumbnailURL,
		CreateVideoParams: videoData.CreateVideoParams,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update the video table", err)
		return
	}

	videoData, err = cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "could not get the video metadata ", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}

func splitMediaType(s string) string {
	ssl := strings.Split(s, "/")
	return ssl[1]
}
