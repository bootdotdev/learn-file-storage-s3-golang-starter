package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
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

	ratio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get aspect ratio from video", err)
		return
	}

	var videoPrefix string
	switch ratio {
	case "16:9":
		videoPrefix = "landscape/"
	case "9:16":
		videoPrefix = "portrait/"
	default:
		videoPrefix = "other/"
	}

	fileName := videoPrefix + getAssetPath(mimeType)

	processPath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fast process video", err)
		return
	}
	defer os.Remove(processPath)

	processedFile, err := os.Open(processPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open process path", err)
		return
	}
	defer processedFile.Close()

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fileName),
		Body:        processedFile,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload to S3", err)
		return
	}

	// videoURL := cfg.getObjectURL(fileName)
	// videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileName) //cfg.getObjectURL(fileName)
	// video.VideoURL = &videoURL

	cdnVideoURL := cfg.getCloudfrontURL(fileName)
	video.VideoURL = &cdnVideoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	// video, err = cfg.dbVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't get signed video from URL", err)
	// 	return
	// }

	respondWithJSON(w, http.StatusOK, video)

}

type VideoStream struct {
	Streams []struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	} `json:"streams"`
}

func getVideoAspectRatio(path string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", path)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe error: %v, details: %s", err, stderr.String())
	}

	videoStream := VideoStream{}
	if err := json.Unmarshal(out.Bytes(), &videoStream); err != nil {
		fmt.Println("Error Unmarshal data")
		return "", err
	}

	width := videoStream.Streams[0].Width
	height := videoStream.Streams[0].Height

	if width == 0 || height == 0 {
		return "", fmt.Errorf("Invalid width/height")
	}

	aspectRatio := getNearestAspectRatio(width, height)

	return aspectRatio, nil
}

// getNearestAspectRatio calculates the simplified ratio and finds the closest standard ratio
func getNearestAspectRatio(width, height int) string {
	// Step 1: Simplify using GCD
	gcd := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}
	divisor := gcd(width, height)
	simplifiedW := width / divisor
	simplifiedH := height / divisor

	// Step 2: Common standard ratios
	standardRatios := map[string][2]int{
		"16:9": {16, 9},
		"9:16": {9, 16},
		"4:3":  {4, 3},
		"3:4":  {3, 4},
		"1:1":  {1, 1},
	}

	// Current ratio in float
	currentRatio := float64(width) / float64(height)

	// Step 3: Find closest match
	closest := ""
	minDiff := math.MaxFloat64
	for name, ratio := range standardRatios {
		ratioVal := float64(ratio[0]) / float64(ratio[1])
		diff := math.Abs(currentRatio - ratioVal)
		if diff < minDiff {
			minDiff = diff
			closest = name
		}
	}

	// Step 4: If difference is small (e.g., less than 0.05), return the standard ratio
	if minDiff < 0.05 {
		return closest
	}

	// Otherwise return the exact simplified ratio
	return fmt.Sprintf("%d:%d", simplifiedW, simplifiedH)
}

// Fast process
func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %s, %v", stderr.String(), err)
	}

	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		return "", fmt.Errorf("Couldn't start file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("file is empty")
	}

	return outputFilePath, nil

}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	objectInput := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	presignReq, err := presignClient.PresignGetObject(context.Background(), &objectInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignReq.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return video, nil
	}

	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) < 2 {
		return video, nil
	}

	signedURL, err := generatePresignedURL(cfg.s3Client, parts[0], parts[1], time.Minute*5)
	if err != nil {
		return video, err
	}
	video.VideoURL = &signedURL

	return video, nil

}
