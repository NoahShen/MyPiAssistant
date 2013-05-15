package main

import (
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/NoahShen/go-xmpp"
	"io/ioutil"
	"logistics"
	"os"
	"pidownloader"
	"runtime"
	"service"
	"speech2text"
	"strings"
	"time"
)

var (
	logConfig  = flag.String("log-config", "", "path of log config file")
	configPath = flag.String("config-path", "", "path of config file")
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	l4g.LoadConfiguration(*logConfig)
	l4g.Debug("MAXPROCS: %d", runtime.GOMAXPROCS(0))
	defer time.Sleep(2 * time.Second) // make sure log4go output the log
	piAssistant := NewPiAssistant()

	piAssistant.ServiceMgr.AddService(&pidownloader.PiDownloader{})
	piAssistant.ServiceMgr.AddService(&logistics.LogisticsService{})

	if initErr := piAssistant.Init(*configPath); initErr != nil { //"....//config/pidownloader.conf"
		l4g.Error("PiAssistant init failed: %v", initErr)
		return
	}
	piAssistant.StartService()
}

type PiAssistant struct {
	xmppClient       *xmpp.XmppClient
	chathandler      xmpp.Handler
	subscribeHandler xmpp.Handler
	stopCh           chan int
	ServiceMgr       *service.ServiceManager
	pushMsgCh        chan *service.PushMessage
	voiceConf        voiceConfig
	xmppConf         xmppConfig
}

func NewPiAssistant() *PiAssistant {
	pi := &PiAssistant{}
	pi.stopCh = make(chan int, 1)
	pi.ServiceMgr = &service.ServiceManager{}
	pi.pushMsgCh = make(chan *service.PushMessage, 10)
	return pi
}

type xmppConfig struct {
	Host       string `json:"host,omitempty"`
	User       string `json:"username,omitempty"`
	Pwd        string `json:"password,omitempty"`
	Master     string `json:"master,omitempty"`
	PingEnable bool   `json:"pingEnable,omitempty"`
}

type voiceConfig struct {
	Confidence float64 `json:"confidence,omitempty"`
}

func (self *PiAssistant) Init(configPath string) error {
	fileData, readErr := ioutil.ReadFile(configPath)
	if readErr != nil {
		l4g.Error("Read config file error: %v", readErr)
		return readErr
	}

	l4g.Info("Loading config file: %s", configPath)
	var configMap map[string]*json.RawMessage
	unmarshalErr := json.Unmarshal(fileData, &configMap)
	if unmarshalErr != nil {
		l4g.Error("Config file formt error: %v", unmarshalErr)
		return unmarshalErr
	}

	var voiceConf voiceConfig
	voiceConfErr := json.Unmarshal(*configMap["voice"], &voiceConf)
	if voiceConfErr != nil {
		l4g.Error("Read voice config error: %v", voiceConfErr)
		return voiceConfErr
	}
	self.voiceConf = voiceConf

	serviceInitErr := self.initServices(configMap)
	if serviceInitErr != nil {
		l4g.Error("Service init failed: %v", serviceInitErr)
		return serviceInitErr
	}
	l4g.Info("Initialize services successful!")

	var xmppConf xmppConfig
	xmppConfigErr := json.Unmarshal(*configMap["xmpp"], &xmppConf)
	if xmppConfigErr != nil {
		l4g.Error("Read xmpp config error: %v", xmppConfigErr)
		return xmppConfigErr
	}
	// connect xmpp server
	connectError := self.connectXmpp(xmppConf)
	if connectError != nil {
		l4g.Error("Connect xmpp server error: %v", connectError)
		return connectError
	}
	self.xmppConf = xmppConf
	l4g.Info("Xmpp is connected!")
	return nil
}

func (self *PiAssistant) connectXmpp(xmppConf xmppConfig) error {
	self.xmppClient = xmpp.NewXmppClient(xmpp.ClientConfig{xmppConf.PingEnable, 3, 30 * time.Second, false, 1})
	xmppErr := self.xmppClient.Connect(xmppConf.Host, xmppConf.User, xmppConf.Pwd)
	return xmppErr
}

func (self *PiAssistant) initServices(configMap map[string]*json.RawMessage) error {
	services := self.ServiceMgr.GetAllServices()
	for _, s := range services {
		initErr := s.Init(configMap[s.GetServiceName()], self.pushMsgCh)
		if initErr != nil {
			return errors.New(fmt.Sprintf("%s init error: %v", s.GetServiceName(), initErr))
		}
	}
	return nil
}

func (self *PiAssistant) StartService() {
	services := self.ServiceMgr.GetAllServices()
	for _, s := range services {
		startErr := s.StartService()
		if startErr != nil {
			l4g.Error("Start service error: %v", startErr)
			return
		}
	}
	l4g.Info("Start services successful!")

	self.chathandler = xmpp.NewChatHandler()
	self.xmppClient.AddHandler(self.chathandler)

	self.subscribeHandler = xmpp.NewSubscribeHandler()
	self.xmppClient.AddHandler(self.subscribeHandler)

	//make sure will receive roster and subscribe message
	self.xmppClient.RequestRoster()
	//make resource available
	self.xmppClient.Send(&xmpp.Presence{})

	stopService := false
	for !stopService {
		select {
		case event := <-self.chathandler.GetEventCh():
			self.handle(event.Stanza.(*xmpp.Message))
		case subsEvent := <-self.subscribeHandler.GetEventCh():
			self.handleSubscribe(subsEvent.Stanza.(*xmpp.Presence))
		case pushMsg := <-self.pushMsgCh:
			self.handlePushMsg(pushMsg)
		case <-self.stopCh:
			stopService = true
		}
	}
}

func (self *PiAssistant) handleSubscribe(subPresence *xmpp.Presence) {
	msg := fmt.Sprintf("%s request to add me as a contact", subPresence.From)
	self.xmppClient.SendChatMessage(self.xmppConf.Master, msg)
}

func (self *PiAssistant) handlePushMsg(pushMsg *service.PushMessage) {
	switch pushMsg.Type {
	case service.Status:
		self.xmppClient.SendPresenceStatus(pushMsg.Message)
	case service.Notification:
		self.xmppClient.SendChatMessage(pushMsg.Username, pushMsg.Message)
	}
}

func (self *PiAssistant) StopService() {
	self.xmppClient.RemoveHandler(self.chathandler)
	self.xmppClient.Disconnect()
	services := self.ServiceMgr.GetAllServices()
	for _, s := range services {
		stopErr := s.Stop()
		if stopErr != nil {
			l4g.Error("Stop service error: %v", stopErr)
		}
	}
	self.stopCh <- 1
}

const (
	voiceMsgPrefix = "Voice IM:"
	fileMsgPrefix  = "I sent you a file through imo:"
)

func (self *PiAssistant) handle(message *xmpp.Message) {
	l4g.Info("Receive message from [%s]: %s", message.From, message.Body)
	command := message.Body

	if strings.HasPrefix(command, voiceMsgPrefix) {
		l4g.Debug("Receive voice message: %s", command)
		voiceUrl := strings.TrimSpace(command[len(voiceMsgPrefix):])
		text, hasConfience, voiceErr := self.convertVoiceToText(voiceUrl)
		if voiceErr != nil {
			l4g.Error("Convert voice to command failed: %v", voiceErr)
			self.xmppClient.SendChatMessage(message.From, "听不清您在说什么，请您再试一下。")
			return
		}
		if !hasConfience {
			msg := fmt.Sprintf("您是不是说[%s]？我听不清，请您再试一下。", text)
			self.xmppClient.SendChatMessage(message.From, msg)
			return
		}
		command = text
	}

	if strings.HasPrefix(command, fileMsgPrefix) {
		l4g.Debug("Receive file command: %s", command)
		fileUrl := strings.TrimSpace(command[len(fileMsgPrefix):])
		filePath, getFileErr := self.getCommandFile(fileUrl)
		defer os.Remove(filePath)
		if getFileErr != nil {
			l4g.Error("Get command file failed: %v", getFileErr)
			msg := "Get command file failed!"
			self.xmppClient.SendChatMessage(message.From, msg)
			return
		}
		command = fmt.Sprintf("file %s %s", fileUrl, filePath)
	}

	command = strings.TrimSpace(command)
	commArr := strings.Split(command, " ")
	l := len(commArr)
	if l == 0 {
		self.xmppClient.SendChatMessage(message.From, "请输入正确的命令！")
		return
	}
	comm := strings.ToLower(commArr[0])
	args := make([]string, 0)
	if l > 1 {
		args = commArr[1:]
	}
	l4g.Debug("Receive command: %s, param: %v", comm, args)
	if comm == "help" {
		helpMsg := self.getHelpMessage()
		self.xmppClient.SendChatMessage(message.From, helpMsg)
		return
	}
	if comm == "subscribed" {
		sender := strings.ToLower(xmpp.ToBareJID(message.From))
		content := ""
		if sender == strings.ToLower(self.xmppConf.Master) {
			subscribed := &xmpp.Presence{
				To:   args[0],
				Type: "subscribed",
			}
			self.xmppClient.Send(subscribed)
			content = fmt.Sprintf("subscribed %s as a contact", args[0])
		} else {
			content = "You are not my master!"
		}
		self.xmppClient.SendChatMessage(message.From, content)
		return
	}
	username := xmpp.ToBareJID(message.From)
	var resp string
	var err error
	findService := false
	services := self.ServiceMgr.GetAllServices()
	for _, s := range services {
		if s.CommandFilter(comm, args) {
			resp, err = s.Handle(username, comm, args)
			findService = true
			break
		}
	}
	var content string
	if !findService {
		content = fmt.Sprintf("命令错误[%s]！请输入\"help\"查询命令！", command)
	} else if err != nil {
		content = err.Error()
	} else {
		content = resp
	}
	self.xmppClient.SendChatMessage(message.From, content)
}

func (self *PiAssistant) getHelpMessage() string {
	helpMessage := "\n"
	services := self.ServiceMgr.GetAllServices()
	for _, s := range services {
		helpMessage = helpMessage +
			fmt.Sprintf("%s 命令:\n%s", s.GetServiceName(), s.GetHelpMessage())
		helpMessage = helpMessage + "------------------\n"
	}
	return helpMessage
}

func (self *PiAssistant) convertVoiceToText(voiceUrl string) (string, bool, error) {
	text, confidence, convertErr := speech2text.Speech2Text(voiceUrl)
	if convertErr != nil {
		return "", false, convertErr
	}
	return text, confidence >= self.voiceConf.Confidence, nil
}
