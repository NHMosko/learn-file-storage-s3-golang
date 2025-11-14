package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, fData, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't FormFile", err)
		return
	}
	defer file.Close()


	mediaType, _, err := mime.ParseMediaType(fData.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't ParseMediaType", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusInternalServerError, "Thumbnail image format not accepted", err)
		return
	}

	fileExtension := strings.Split(mediaType, "/")[1]

	var randomness = make([]byte, 32)
	rand.Read(randomness)
	thisRand := base64.RawURLEncoding.EncodeToString(randomness)


	tnPath := fmt.Sprintf("/%s.%s", thisRand, fileExtension)
	newFile, err := os.Create(filepath.Join(cfg.assetsRoot, tnPath))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create new file", err)
		return
	}

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy data to file", err)
		return
	}

	tnURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, thisRand, fileExtension)
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't GetVideo", err)
		return
	}
	video.ThumbnailURL = &tnURL

	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't UpdateVideo", err)
		return
	}

	fmt.Println("upload complete.")
	respondWithJSON(w, http.StatusOK, video)
}
