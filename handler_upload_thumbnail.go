package main

import (
	"fmt"
	"net/http"

	"io"

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

	data, header, erro := r.FormFile("thumbnail")
	if erro != nil {
		respondWithError(w, http.StatusBadRequest, "couldnt form file", erro)
		return
	}
	mediaType := header.Header.Get("Content-Type")
	bytedata, errs := io.ReadAll(data)
	if errs != nil {
		respondWithError(w, http.StatusBadRequest, "couldnt read ma", errs)
		return
	}
	videodeets, errz := cfg.db.GetVideo(videoID)
	if errz != nil {
		respondWithError(w, http.StatusBadRequest, "couldnt fetch frm db", errz)
		return
	}
	id := videodeets.UserID

	if id != userID {
		respondWithError(w, http.StatusUnauthorized, "could not validate", err)
		return

	}
	newthumnail := thumbnail{
		data:      bytedata,
		mediaType: mediaType,
	}

	videoThumbnails[videoID] = newthumnail
	str := fmt.Sprintf("http://localhost:8091/api/thumbnails/%s", videoID)
	videodeets.ThumbnailURL = &str
	cfg.db.UpdateVideo(videodeets)

	respondWithJSON(w, http.StatusOK, videodeets)

}
