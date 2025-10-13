package main

import (
	"testing"
)

func TestGetVideoAspectRatio(t *testing.T) {
	filepath := "samples/boots-video-horizontal.mp4"
	expOut := "16:9"
	output, err := getVideoAspectRatio(filepath)
	if err != nil {
		t.Errorf("error running the command: %s", err)
	}
	if output != expOut {
		t.Errorf("expected output: %s, actual: %s", expOut, output)
	}

	filepath = "samples/boots-video-vertical.mp4"
	expOut = "9:16"
	output, err = getVideoAspectRatio(filepath)
	if err != nil {
		t.Errorf("error running the command: %s", err)
	}
	if output != expOut {
		t.Errorf("expected output: %s, actual: %s", expOut, output)
	}
}
