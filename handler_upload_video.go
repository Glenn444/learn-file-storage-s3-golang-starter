package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	const maxMemory = 1 << 30
	r.ParseMultipartForm(maxMemory)

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	videoMeta, err := cfg.db.GetVideo(videoID)
	if userID != videoMeta.UserID {
		respondWithError(w, http.StatusUnauthorized, "Video is not the users", err)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	parts := strings.Split(contentType, "/")

	if contentType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Not a video", err)
		return
	}

	fileCreated, errFile := os.CreateTemp(".", "tubely-upload.mp4")
	if errFile != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating tempfile", err)
		return
	}
	defer os.Remove("tubely-upload.mp4")
	defer fileCreated.Close()

	if _, err := io.Copy(fileCreated, file); err != nil {
		log.Fatal(err)
	}

	key := make([]byte, 16)
	rand.Read(key)

	hexKey := hex.EncodeToString(key)

	videoKey := fmt.Sprintf("%s.%s", hexKey, parts[1])
	fmt.Printf("hex key: %v\n", videoKey)
	fileCreated.Seek(0, io.SeekStart) //allows us to read the file again from beginning

	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &videoKey,
		Body:        fileCreated,
		ContentType: &contentType,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading video to s3 bucket", err)
		return
	}

	videoUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, videoKey)
	fmt.Printf("videoUrl: %s\n", videoUrl)
	videoMeta.VideoURL = &videoUrl
	cfg.db.UpdateVideo(videoMeta)

	respondWithJSON(w, http.StatusOK, videoMeta)
}
