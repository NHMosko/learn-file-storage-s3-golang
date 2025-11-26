package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
)

type StreamsStruct struct {
	Streams []struct {
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	buff := new(bytes.Buffer)
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = buff 

	err := cmd.Run()
	if err != nil {
		log.Printf("Executing command failed: %v", cmd)
		return "", err
	}

	ratio := StreamsStruct{}

	if err := json.Unmarshal(buff.Bytes(), &ratio); err != nil {
		return "", err
	}

	stream := ratio.Streams[0]
	if stream.Width / 16 == stream.Height / 9 {
		return "landscape", nil
	} else if stream.Width / 9 == stream.Height / 16 {
		return "portrait", nil
	}

	return "other", nil
}
