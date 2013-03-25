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

func TestGetActive(t *testing.T) {
	keys := []string{"gid", "status"}
	reply, err := GetActive(keys)
	if err != nil {
		log.Fatal("GetActive", err)
	}
	log.Println(reply)

	reply2, err2 := GetActive(nil)
	if err2 != nil {
		log.Fatal("GetActive", err2)
	}
	log.Println(reply2)
}
