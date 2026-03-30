package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	// provided code
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
	// End provided code

	// Bit-shifted the number 10 to the left by 20 places to get an int that stores the proper number of bytes.
	// Bit shifting is a way to multiply by powers of 2. 10 << 20 is the same as 10 * 1024 * 1024, which is 10 MB.
	const maxMemory = 10 << 20 // 10 MB (megabytes)

	// Parse the form data
	r.ParseMultipartForm(maxMemory)

	// Get the image data from the form
	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer fileData.Close()

	mediaType := fileHeaders.Header.Get("Content-Type")
	temp, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse media type", err)
		return
	}
	
	var fileExtension string
	if temp == "image/png" {
		fileExtension = "png"
	} else if temp == "image/jpeg" {
		fileExtension = "jpeg"
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating random slice", err)
		return
	}

	thumbnailID := base64.RawURLEncoding.EncodeToString(key)
	
	fmt.Printf("mediaType=%v\n", mediaType)
	fmt.Printf("fileExtension=%v\n", fileExtension)

	assetPath := fmt.Sprintf("%v.%v", thumbnailID, fileExtension)

	file, err := os.Create(cfg.assetsRoot + "/" + assetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating file path", err)
		return
	}

	defer file.Close()

	_, err = io.Copy(file, fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error copying/writing file data to disk", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error grabbing video metadata from database", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "authenticated user is not the video owner", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%v/assets/%v", os.Getenv("PORT"), assetPath)
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "handlerUploadThumbnail - error updating video entry in db", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
