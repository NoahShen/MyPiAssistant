package main

import (
	"aqi"
	l4g "code.google.com/p/log4go"
	"foodprice"
	"github.com/NoahShen/go-simsimi"
	"github.com/NoahShen/go-xmpp"
	"logistics"
	"pidownloader"
	"runtime"
	"speech2text"
	"testing"
	"time"
)

func TestPiAssistant(t *testing.T) {
	xmpp.Debug = false
	aqi.Debug = true
	logistics.Debug = true
	simsimi.Debug = true
	speech2text.Debug = true
	foodprice.Debug = true
	runtime.GOMAXPROCS(runtime.NumCPU())
	l4g.LoadConfiguration("../../config/logConfig.xml")
	defer time.Sleep(2 * time.Second)
	piAssistant := NewPiAssistant()
	piAssistant.ServiceMgr.AddService(&pidownloader.PiDownloader{})
	piAssistant.ServiceMgr.AddService(&logistics.LogisticsService{})
	piAssistant.ServiceMgr.AddService(&aqi.AqiService{})
	piAssistant.ServiceMgr.AddService(&foodprice.FoodPriceService{})

	if initErr := piAssistant.Init("../../config/piassistant.conf"); initErr != nil {
		t.Fatal("init error:", initErr)
	}
	piAssistant.StartService()
}
