package main

import (
	"aqi"
	l4g "code.google.com/p/log4go"
	"github.com/NoahShen/go-simsimi"
	"github.com/NoahShen/go-xmpp"
	"logistics"
	"pidownloader"
	"runtime"
	"testing"
	"time"
)

func TestPiAssistant(t *testing.T) {
	xmpp.Debug = true
	aqi.Debug = true
	logistics.Debug = true
	simsimi.Debug = true
	runtime.GOMAXPROCS(runtime.NumCPU())
	l4g.LoadConfiguration("../../config/logConfig.xml")
	defer time.Sleep(2 * time.Second)
	piAssistant := NewPiAssistant()
	piAssistant.ServiceMgr.AddService(&pidownloader.PiDownloader{})
	piAssistant.ServiceMgr.AddService(&logistics.LogisticsService{})
	piAssistant.ServiceMgr.AddService(&aqi.AqiService{})

	if initErr := piAssistant.Init("../../config/piassistant.conf"); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piAssistant.StartService()
}
