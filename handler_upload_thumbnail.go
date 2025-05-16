package main

import (
	"fmt"
	"io"
	"log"
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
		respondWithError(w, 400, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 400, "Error parsing media type from request", err)
		return
	}

	if mediaType == "image/jpeg" || mediaType == "image/png" {
		log.Println("Image found in Content-Type header of HTTP request.")
	} else {
		respondWithError(w, 400, "Media type is not 'image/jpeg' or 'image/png'", nil)
		return
	}

	fileName := fmt.Sprintf("%s.%s", videoIDString, strings.Replace(header.Header.Get("Content-Type"), "image/", "", -1))
	filePath := filepath.Join(".", cfg.assetsRoot, fileName)
	osFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, 500, "Issue creating thumbnail file on fileserver", err)
		return
	}

	_, err = io.Copy(osFile, file)
	if err != nil {
		respondWithError(w, 500, "Issue copying raw bytes to thumbnail file on fileserver", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 404, "Video not found", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, 401, "Authenticated user is not the owner of the video", nil)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:8091/assets/%s", fileName)
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, 500, "Unable to update video metadata in database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
