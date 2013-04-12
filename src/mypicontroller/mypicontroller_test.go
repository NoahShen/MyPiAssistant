package main

import (
	l4g "code.google.com/p/log4go"
	"runtime"
	"testing"
)

func TestPiDownloader(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
