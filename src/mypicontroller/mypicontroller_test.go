package main

import (
	//"log"
	"testing"
)

func TestPiDownloader(t *testing.T) {
	piController, err := NewPiController("../config/pidownloader.conf")
	if err != nil {
		t.Fatal("start error:", err)
	}

	if initErr := piController.Init(); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piController.StartService()
}
