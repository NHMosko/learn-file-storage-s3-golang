package main

import (
	"log"
	"os/exec"
)

func processVideoForFastStart(filepath string) (string, error) {
	outpath := filepath + ".processing"
	cmd := exec.Command(
		"ffmpeg",
		"-i",
		filepath,
		"-c",
		"copy",
		"-movflags",
		"faststart",
		"-f",
		"mp4",
		outpath,
	)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to run command: %s", cmd)
		return "", err
	}

	return outpath, nil
}
