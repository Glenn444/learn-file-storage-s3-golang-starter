package main

import (
	"fmt"
	"io"
	"log"
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


	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file,fileHeader,err := r.FormFile("thumbnail")
	if err != nil{
		respondWithError(w,http.StatusBadRequest,"Unable to parse form file",err)
		return
	}
	defer file.Close()

	videoMeta,err := cfg.db.GetVideo(videoID)
	if userID != videoMeta.UserID{
		respondWithError(w,http.StatusUnauthorized,"Video is not the users",err)
		return
	}
	
	contentType := fileHeader.Header.Get("Content-Type")
	parts := strings.Split(contentType, "/")

	if !(strings.HasPrefix(contentType, "image/")){
		respondWithError(w,http.StatusBadRequest,"Not an image",err)
		return
	}

	imageUrl := fmt.Sprintf("%s.%s",videoID,parts[1])
	assetsPath := filepath.Join(cfg.assetsRoot,imageUrl)
	
	fileCreated,errFile := os.Create(assetsPath)
	if errFile != nil{
		respondWithError(w,http.StatusInternalServerError,"Error creating file",err)
		return
	}
	if _, err := io.Copy(fileCreated,file);err != nil{
		log.Fatal(err)
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s",cfg.port,imageUrl)
	
	videoMeta.ThumbnailURL = &thumbnailUrl

	cfg.db.UpdateVideo(videoMeta)

	respondWithJSON(w, http.StatusOK, videoMeta)
}
