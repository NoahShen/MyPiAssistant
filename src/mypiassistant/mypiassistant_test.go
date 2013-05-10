package main

import (
	l4g "code.google.com/p/log4go"
	"github.com/NoahShen/go-xmpp"
	"logistics"
	"pidownloader"
	"runtime"
	"testing"
	"time"
)

func TestPiDownloader(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	l4g.LoadConfiguration("../../config/logConfig.xml")
	defer time.Sleep(2 * time.Second)
	piAssistant := NewPiAssistant()
	piAssistant.ServiceMgr.AddService(&pidownloader.PiDownloader{})
	piAssistant.ServiceMgr.AddService(&logistics.LogisticsService{})
	xmpp.Debug = true
	if initErr := piAssistant.Init("../../config/piassistant.conf"); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piAssistant.StartService()
}
