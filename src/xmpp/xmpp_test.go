package xmpp

import (
	"fmt"
	"testing"
	"time"
)

func NoTestLogin(t *testing.T) {
	var server = "talk.google.com:443"
	var username = "NoahPi87@gmail.com"
	var password = "15935787"
	talk, err := NewClient(server, username, password)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			chat, err := talk.Recv()
			if err != nil {
				t.Fatal(err)
			}
			switch v := chat.(type) {
			case Chat:
				fmt.Println(v.Remote, v.Text)
			case Presence:
				fmt.Println(v.From, v.Show)
			}
		}
	}()

	time.Sleep(30 * time.Second)
	talk.Close()
}
