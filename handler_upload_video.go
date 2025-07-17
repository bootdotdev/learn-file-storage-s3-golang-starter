package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse multipart form dfata", err)
		return
	}

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video id is not valid", err)
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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized access to the video", nil)
		return
	}

	fmt.Println("uploading video with id: ", videoID, "by user: ", userID)

	file, handler, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse the video", err)
		return
	}
	defer file.Close()

	mimeType, _, err := mime.ParseMediaType(handler.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse mime type for video", err)
		return
	}

	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Video must be mp4 file", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubley-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy to tmp file", err)
		return
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seed through video", err)
		return
	}

	fileName := getAssetPath(mimeType)

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fileName),
		Body:        tmpFile,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload to S3", err)
		return
	}
	videoURL := cfg.getObjectURL(fileName)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)

}
