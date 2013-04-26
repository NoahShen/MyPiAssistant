package main

import (
	l4g "code.google.com/p/log4go"
	"runtime"
	"testing"
	"time"
)

func TestPiDownloader(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l4g.LoadConfiguration("../config/logConfig.xml")
	defer time.Sleep(2 * time.Second)
	piAssistant, err := NewPiAssistant("../config/piassistant.conf")
	if err != nil {
		t.Fatal("start error:", err)
		return
	}

	if initErr := piAssistant.Init(); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piAssistant.StartService()
}
