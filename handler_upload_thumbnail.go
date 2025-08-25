package main

import (
	"fmt"
	"io"
	"net/http"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file,_,err := r.FormFile("thumbnail")
	if err != nil{
		respondWithError(w,http.StatusBadRequest,"Unable to parse form file",err)
		return
	}
	defer file.Close()

	fileInfo,err := io.ReadAll(file)
	if err != nil{
		respondWithError(w,http.StatusInternalServerError,"Failed to read from file",err)
		return
	}
	videoMeta,err := cfg.db.GetVideo(videoID)
	if userID != videoMeta.UserID{
		respondWithError(w,http.StatusUnauthorized,"Video is not the users",err)
		return
	}
	newThumbnail := thumbnail{
		data: fileInfo,
		mediaType: "image",
	}

	videoThumbnails[videoMeta.ID] = newThumbnail
	port := cfg.port

	url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", port, videoID)

	videoMeta.ThumbnailURL = &url

	cfg.db.UpdateVideo(videoMeta)

	respondWithJSON(w, http.StatusOK, videoMeta)
}
