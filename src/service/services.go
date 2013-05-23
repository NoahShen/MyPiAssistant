package service

import (
	"encoding/json"
)

type Service interface {
	GetServiceId() string
	GetServiceName() string
	Init(*json.RawMessage, chan<- *PushMessage) error
	StartService() error
	IsStarted() bool
	Stop() error
	GetHelpMessage() string
	CommandFilter(string, []string) bool
	Handle(string, string, []string) (string, error)
}

type MessageType string

const (
	Status       = MessageType("Status")
	Notification = MessageType("Notification")
)

type PushMessage struct {
	Type     MessageType
	Username string
	Message  string
}
