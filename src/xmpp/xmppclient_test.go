package xmpp

import (
	"fmt"
	"testing"
)

var server = "talk.google.com:443"
var username = "NoahPi87@gmail.com"
var password = "15935787"

func TestSendMessage(t *testing.T) {
	xmppClient := NewXmppClient()
	err := xmppClient.Connect(server, username, password)
	if err != nil {
		t.Fatal(err)
	}

	chathandler := NewChatHandler()
	xmppClient.AddHandler(chathandler)
	for msg := range chathandler.GetHandleCh() {
		fmt.Println("chat message:", msg)
		chatMessage := msg.(Chat)
		replyChat := &Chat{chatMessage.Remote, chatMessage.Type, "echo:" + chatMessage.Text}
		xmppClient.Send(replyChat)
	}

}
