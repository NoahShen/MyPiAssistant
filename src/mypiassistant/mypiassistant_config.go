package main

import (
	"encoding/json"
)

type XmppConfig struct {
	Host            string `json:"host,omitempty"`
	User            string `json:"username,omitempty"`
	Pwd             string `json:"password,omitempty"`
	Master          string `json:"master,omitempty"`
	PingEnable      bool   `json:"pingEnable,omitempty"`
	ReconnectEnable bool   `json:"reconnectEnable,omitempty"`
}

type VoiceConfig struct {
	Confidence float64 `json:"confidence,omitempty"`
}

type ServiceConfig struct {
	ServiceId string           `json:"serviceId,omitempty"`
	Autostart bool             `json:"autostart,omitempty"`
	ConfigRaw *json.RawMessage `json:"config,omitempty"`
}

type PiAssistantConfig struct {
	XmppConf       *XmppConfig     `json:"xmpp,omitempty"`
	VoiceConf      *VoiceConfig    `json:"voice,omitempty"`
	ServicesConfig []ServiceConfig `json:"services,omitempty"`
}
