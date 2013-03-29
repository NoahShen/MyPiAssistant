package main

import (
	"flag"
	"fmt"
	"pidownloader"
)

var (
	configpath = flag.String("configpath", "", "path of config file")
)

func main() {
	flag.Parse()
	piDer, err := pidownloader.NewPidownloader(*configpath) //"../config/pidownloader.conf"
	if err != nil {
		fmt.Println("start error:", err)
		return
	}

	if initErr := piDer.Init(); initErr != nil {
		fmt.Println("init error:", initErr)
		return
	}
	piDer.StartService()
}
