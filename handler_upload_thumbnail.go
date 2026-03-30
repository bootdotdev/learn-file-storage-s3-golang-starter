package main

import (
	"io"
	"os"
	"fmt"
	"net/http"

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
	fmt.Printf("mediaType=%v\n", mediaType)

	// Log file details (filename, size, MIME header)
	// log.Printf("Uploaded File: %+v\n", header.Filename)
	// log.Printf("File Size: %+v\n", header.Size)
	// log.Printf("MIME Header: %+v\n", header.Header)

	// Read the image data into a byte slice
	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading file data", err)
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

	newThumbnail := thumbnail{
		data: fileBytes,
		mediaType: mediaType,
	}

	videoThumbnails[videoID] = newThumbnail

	// Update the video metadata so that it has a new thumbnail URL
	thumbnailURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", os.Getenv("PORT"), videoID) 
	videoMetadata.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "handlerUploadThumbnail - error updating video entry in db", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
