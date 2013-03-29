package xmpp

import (
	"log"
	"sync"
)

type XmppClient struct {
	client     *Client
	sendQueue  chan interface{}
	stopSendCh chan int
	mutex      sync.Mutex
	handlers   []Handler
}

func NewXmppClient() *XmppClient {
	xmppClient := new(XmppClient)
	xmppClient.sendQueue = make(chan interface{}, 10)
	xmppClient.stopSendCh = make(chan int)
	return xmppClient
}

func (self *XmppClient) Connect(host, username, password string) error {
	client, err := NewClient(host, username, password)
	if err != nil {
		return err
	}
	self.client = client
	go self.startSendMessage()
	go self.startReadMessage()
	return nil
}

func (self *XmppClient) Disconnect() error {
	self.stopSendCh <- 1
	return self.client.Close()
}

func (self *XmppClient) Send(msg interface{}) {
	self.sendQueue <- msg
}

func (self *XmppClient) startSendMessage() {
	for {
		select {
		case msg := <-self.sendQueue:
			self.client.Send(msg)
		case <-self.stopSendCh:
			close(self.sendQueue)
			break
		}
	}
}

func (self *XmppClient) startReadMessage() {
	for {
		msg, err := self.client.Recv()
		if err != nil {
			log.Println("Read message error:", err)
			break
		}
		self.fireHandler(msg)
	}
}

func (self *XmppClient) AddHandler(handler Handler) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.handlers = append(self.handlers, handler)
}

func (self *XmppClient) RemoveHandler(handler Handler) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for i, oldHandler := range self.handlers {
		if oldHandler == handler {
			self.handlers = append(self.handlers[0:i], self.handlers[i+1:]...)
			break
		}
	}
}

func (self *XmppClient) RemoveHandlerByIndex(i int) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.handlers = append(self.handlers[0:i], self.handlers[i+1:]...)
}

func (self *XmppClient) fireHandler(msg interface{}) {
	for i := len(self.handlers) - 1; i >= 0; i-- {
		h := self.handlers[i]
		if h.Filter(msg) {
			h.GetHandleCh() <- msg
			if h.IsOnce() {
				self.RemoveHandlerByIndex(i)
			}
		}
	}
}
