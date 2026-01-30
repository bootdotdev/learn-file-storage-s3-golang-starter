package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	data, header, err := r.FormFile("headers")
	mediaType := header.Header.Get("Content-Type")
	bytedata, err := io.ReadAll(data)
	videodeets, err := cfg.db.GetVideo(videoID)
	id := videodeets.UserID

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "token not found", err)
		return
	}
	ids, errm := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "could not validate", errm)
		return
	}
	if ids != id {
		respondWithError(w, http.StatusUnauthorized, "could not validate", errm)
		return

	}
	newthumnail := thumbnail{
		data:      bytedata,
		mediaType: mediaType,
	}
	videoThumbnails[videoID] = newthumnail

	tn, ok := videoThumbnails[videoID]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
		return
	}

	w.Header().Set("Content-Type", tn.mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(tn.data)))

	_, err = w.Write(tn.data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}
