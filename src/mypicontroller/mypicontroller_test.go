package main

import (
	l4g "log4go"
	"testing"
)

func TestPiDownloader(t *testing.T) {
	l4g.LoadConfiguration("../config/logConfig.xml")
	piController, err := NewPiController("../config/pidownloader.conf")
	if err != nil {
		t.Fatal("start error:", err)
		return
	}

	if initErr := piController.Init(); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piController.StartService()
}
