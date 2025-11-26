package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	http.MaxBytesReader(w, r.Body, 1 << 30)
	defer r.Body.Close()

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

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video from Database", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", err)
		return
	}

	file, fData, err :=	r.FormFile("video")
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
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusInternalServerError, "Video format not accepted", err)
		return
	}
	fileExtension := strings.Split(mediaType, "/")[1]

	temp, err := os.CreateTemp("", "temp-video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create temporary file", err)
		return
	}

	defer os.Remove(temp.Name())
	defer temp.Close()

	_, err = io.Copy(temp, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy data to file", err)
		return
	}

	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't return pointer to beggining", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(temp.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get aspect ratio", err)
		return
	}

	fastStart, err := processVideoForFastStart(temp.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process for fast start", err)
		return
	}

	f, err := os.Open(fastStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to open processed file", err)
		return
	}
	defer os.Remove(fastStart)
	defer f.Close()
	stat, _ := f.Stat()



	var randomness = make([]byte, 32)
	rand.Read(randomness)
	thisRand := base64.RawURLEncoding.EncodeToString(randomness)

	fileKey := fmt.Sprintf("%s/%s.%s", aspectRatio, thisRand, fileExtension)

	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(fileKey),
		Body: f,
		ContentType: aws.String(mediaType),
		ContentLength: aws.Int64(stat.Size()),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to put object on s3", err)
		return
	}

	newURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, fileKey)
	video.VideoURL = &newURL

	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't UpdateVideo", err)
		return
	}

	fmt.Println("upload complete.")
	respondWithJSON(w, http.StatusOK, video)
}
