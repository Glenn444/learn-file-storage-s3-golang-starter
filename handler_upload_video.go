package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
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

	tempfile, errFile := os.CreateTemp("", "tubely-upload.mp4")
	if errFile != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating tempfile", err)
		return
	}
	defer os.Remove(tempfile.Name())
	defer tempfile.Close()

	if _, err := io.Copy(tempfile, file); err != nil {
		respondWithError(w,http.StatusInternalServerError,"error occurred copying the files",err)
		return
	}

	
	key := make([]byte, 16)
	rand.Read(key)

	hexKey := hex.EncodeToString(key)

	videoKey := fmt.Sprintf("%s.%s", hexKey, parts[1])
	
	_,err = tempfile.Seek(0, io.SeekStart) //allows us to read the file again from beginning
	if err != nil{
		respondWithError(w,http.StatusInternalServerError,"could not reset file pointer",err)
		return
	}

	directory := ""
	aspectRatio,err := getVideoAspectRatio(tempfile.Name())
	if err != nil{
		
		respondWithError(w,http.StatusInternalServerError,"error occurred getting aspect ratio",err)
		return
	}

	switch aspectRatio {
	case "16:9":
		directory = "landscape"
	case "9:16":
		directory = "portrait"
	default:
		directory = "other"
	}

	newVideoKey := path.Join(directory,videoKey)
	processedFilePath,err := processVideoForFastStart(tempfile.Name())
	if err != nil{
		respondWithError(w,http.StatusInternalServerError,"Error Processing video for faster start",err)
		return
	}
	defer os.Remove(processedFilePath)

	processedFile,err := os.Open(processedFilePath)

	if err != nil{
		respondWithError(w,http.StatusInternalServerError,"Could not open processed file",err)
		return
	}

	defer processedFile.Close()


	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &newVideoKey,
		Body:        processedFile,
		ContentType: &contentType,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading video to s3 bucket", err)
		return
	}

	videoUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, newVideoKey)
	
	videoMeta.VideoURL = &videoUrl
	cfg.db.UpdateVideo(videoMeta)

	respondWithJSON(w, http.StatusOK, videoMeta)
}
