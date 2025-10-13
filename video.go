package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filepath string) (string, error) {
	type vidAspectJSON struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)

	output := bytes.Buffer{}
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	vidAspect := vidAspectJSON{}

	err = json.Unmarshal(output.Bytes(), &vidAspect)
	if err != nil {
		return "", err
	}

	w := vidAspect.Streams[0].Width
	h := vidAspect.Streams[0].Height

	if w/16 == h/9 {
		return "16:9", nil
	} else if w/9 == h/16 {
		return "9:16", nil
	} else if w/16 == h/10 {
		return "16:10", nil
	} else if w/10 == h/16 {
		return "10:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := fmt.Sprintf("%s.processing", filePath)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputPath, nil
}
