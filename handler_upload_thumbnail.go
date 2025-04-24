package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

	filedata, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not read the file data", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "the video ID does not exists", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorised", err)
		return
	}

	fileString := base64.StdEncoding.EncodeToString(filedata)

	thumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, fileString)

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
