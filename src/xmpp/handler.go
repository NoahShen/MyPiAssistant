package xmpp

import ()

type Handler interface {
	GetHandleCh() chan interface{}
	Filter(interface{}) bool
	IsOnce() bool
}

type ChatHandler struct {
	handlCh chan interface{}
}

func NewChatHandler() Handler {
	return &ChatHandler{make(chan interface{})}
}

func (self *ChatHandler) GetHandleCh() chan interface{} {
	return self.handlCh
}

func (self *ChatHandler) Filter(msg interface{}) bool {
	switch chat := msg.(type) {
	case Chat:
		return chat.Text != ""
	}
	return false
}

func (self *ChatHandler) IsOnce() bool {
	return false
}
