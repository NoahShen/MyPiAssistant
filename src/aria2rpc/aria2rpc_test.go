package aria2rpc

import (
	"log"
	"testing"
)

func NoTestAddUri(t *testing.T) {
	uri := "https://www.kernel.org/pub/linux/kernel/v3.x/linux-3.8.4.tar.xz"
	//uri := "http://bt.ktxp.com/torrents/2013/03/24/3a091e44394e2ec6345cc263accf31058eda504e.torrent"

	params := make(map[string]string)
	params["max-download-limit"] = "1K"
	gid, err := AddUri(uri, params)
	if err != nil {
		log.Fatal("add Uri error:", err)
	}
	log.Println(gid)
}

func NoTestAddTorrent(t *testing.T) {
	gid, err := AddTorrent("/home/noah/DueWest.torrent")
	if err != nil {
		log.Fatal("add Torrent error:", err)
	}
	log.Println(gid)
}

func NoTestRemove(t *testing.T) {
	gid, err := Remove("12", true)
	if err != nil {
		log.Fatal("Remove error:", err)
	}
	log.Println(gid)
}

func NoTestPause(t *testing.T) {
	gid, err := Pause("16", false)
	if err != nil {
		log.Fatal("Pause error:", err)
	}
	log.Println(gid)
}

func NoTestUnpause(t *testing.T) {
	gid, err := Unpause("16")
	if err != nil {
		log.Fatal("Unpause error:", err)
	}
	log.Println(gid)
}

func TestGetStatus(t *testing.T) {
	keys := []string{"gid", "status"}
	reply, err := GetStatus("16", keys)
	if err != nil {
		log.Fatal("GetStatus", err)
	}
	log.Println(reply)
}

func NoTestGetActive(t *testing.T) {
	//keys := []string{"gid", "status"}
	//reply, err := GetActive(keys)
	//if err != nil {
	//	log.Fatal("GetActive", err)
	//}
	//log.Println(reply)

	reply2, err2 := GetActive(nil)
	if err2 != nil {
		log.Fatal("GetActive", err2)
	}
	log.Println(reply2)
}

func TestGetWaiting(t *testing.T) {
	keys := []string{"gid", "status"}
	reply, err := GetWaiting(0, 10, keys)
	if err != nil {
		log.Fatal("Waiting error:", err)
	}
	log.Println(reply)
}

func NoTestGetStopped(t *testing.T) {
	reply, err := GetStopped(0, 10, nil)
	if err != nil {
		log.Fatal("Waiting error:", err)
	}
	log.Println(reply)
}

func NoTestGetOption(t *testing.T) {
	reply, err := GetOption("16")
	if err != nil {
		log.Fatal("GetOption error:", err)
	}
	log.Println(reply)
}

func NoTestChangeOption(t *testing.T) {
	params := make(map[string]string)
	params["max-download-limit"] = "7K"
	reply, err := ChangeOption("16", params)
	if err != nil {
		log.Fatal("ChangeOption error:", err)
	}
	log.Println(reply)
}

func NoTestGetGlobalStat(t *testing.T) {
	reply, err := GetGlobalStat()
	if err != nil {
		log.Fatal("GetGlobalStat error:", err)
	}
	log.Println(reply)
}
