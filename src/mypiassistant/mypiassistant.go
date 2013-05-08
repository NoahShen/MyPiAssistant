package main

import (
	"code.google.com/p/goconf/conf"
	l4g "code.google.com/p/log4go"
	"errors"
	"flag"
	"fmt"
	"github.com/NoahShen/go-xmpp"
	"github.com/robfig/cron"
	"logistics"
	"os"
	"pidownloader"
	"runtime"
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
	defer time.Sleep(2 * time.Second)               // make sure log4go output the log
	piAssistant, err := NewPiAssistant(*configPath) //"../config/pidownloader.conf"
	if err != nil {
		return
	}

	if initErr := piAssistant.Init(); initErr != nil {
		return
	}
	piAssistant.StartService()
}

type config struct {
	StatUpdateCron      string
	RpcUrl              string
	RpcVersion          string
	XmppHost            string
	XmppUser            string
	XmppPwd             string
	Gdriveid            string
	TorrentDir          string
	Confidence          float64
	LogisticsDbFile     string
	LogisticsUpdateCron string
	BeforeLastUpdate    int
}

type PiAssistant struct {
	piDownloader     *pidownloader.PiDownloader
	xmppClient       *xmpp.XmppClient
	config           *config
	stopCh           chan int
	chathandler      xmpp.Handler
	logisticsService *logistics.LogisticsService
	cron             *cron.Cron
}

func loadConfig(configPath string) (*config, error) {
	var c *conf.ConfigFile
	var err error
	c, err = conf.ReadConfigFile(configPath)
	if err != nil {
		return nil, err
	}
	config := new(config)
	if config.XmppHost, err = c.GetString("xmpp", "host"); err != nil {
		return nil, err
	}

	if config.XmppUser, err = c.GetString("xmpp", "username"); err != nil {
		return nil, err
	}

	if config.XmppPwd, err = c.GetString("xmpp", "password"); err != nil {
		return nil, err
	}

	if config.Confidence, err = c.GetFloat64("voice", "confidence"); err != nil {
		return nil, err
	}

	if config.RpcUrl, err = c.GetString("aria2", "rpc_url"); err != nil {
		return nil, err
	}

	if config.RpcVersion, err = c.GetString("aria2", "rpc_version"); err != nil {
		return nil, err
	}
	if config.StatUpdateCron, err = c.GetString("aria2", "stat_update_cron"); err != nil {
		return nil, err
	}
	if config.TorrentDir, err = c.GetString("aria2", "torrent_dir"); err != nil {
		return nil, err
	}
	if config.Gdriveid, err = c.GetString("aria2", "gdriveid"); err != nil {
		return nil, err
	}
	if config.LogisticsDbFile, err = c.GetString("logistics", "db_file"); err != nil {
		return nil, err
	}
	if config.LogisticsUpdateCron, err = c.GetString("logistics", "logistics_update_cron"); err != nil {
		return nil, err
	}

	if config.BeforeLastUpdate, err = c.GetInt("logistics", "before_last_time"); err != nil {
		return nil, err
	}

	return config, nil
}

func NewPiAssistant(configPath string) (*PiAssistant, error) {
	l4g.Info("Loading config file: %s", configPath)
	config, configErr := loadConfig(configPath)
	if configErr != nil {
		l4g.Error("Load config error: %v", configErr)
		return nil, configErr
	}
	piDer, err := pidownloader.NewPidownloader(config.RpcUrl, config.Gdriveid, config.TorrentDir)
	if err != nil {
		l4g.Error("Create PiDownloader error: %v", err)
		return nil, err
	}
	logisticsService, logiErr :=
		logistics.NewLogisticsService(config.LogisticsDbFile, int64(config.BeforeLastUpdate))
	if logiErr != nil {
		l4g.Error("Create LogisticService error: %v", logiErr)
		return nil, err
	}

	pi := &PiAssistant{}
	pi.config = config
	pi.piDownloader = piDer
	pi.xmppClient = xmpp.NewXmppClient()
	pi.logisticsService = logisticsService
	pi.cron = cron.New()
	return pi, nil
}

func (self *PiAssistant) Init() error {
	_, statErr := self.piDownloader.Process("", "getstat")
	if statErr != nil {
		l4g.Error("Call aria2 error: %v", statErr)
		return statErr
	}
	l4g.Info("Aria2 is running!")
	// connect xmpp server
	xmppErr := self.xmppClient.Connect(self.config.XmppHost, self.config.XmppUser, self.config.XmppPwd)
	if xmppErr != nil {
		l4g.Error("Connect xmpp server error: %v", xmppErr)
		return xmppErr
	}
	l4g.Info("Xmpp is connected!")
	self.cron.AddFunc(self.config.StatUpdateCron, func() {
		self.updateDownloadStat()
	})

	self.cron.AddFunc(self.config.LogisticsUpdateCron, func() {
		self.updateLogistics()
	})
	return nil
}

func (self *PiAssistant) updateDownloadStat() {
	status, statErr := self.piDownloader.Process("", "getstat")
	if statErr != nil {
		l4g.Error("Get download stat error: %v", statErr)
		self.xmppClient.SendPresenceStatus(statErr.Error())
	} else {
		self.xmppClient.SendPresenceStatus(status)
	}
}

func (self *PiAssistant) updateLogistics() {
	logisticsCh := make(chan *logistics.ChangedLogisticsInfo, 10)
	go self.logisticsService.UpdateAndGetChangedLogistics(logisticsCh)
	for changedInfo := range logisticsCh {
		progress := self.logisticsService.FormatLogiOutput(changedInfo.NewRecords)
		messageContent := fmt.Sprintf("\n[%s] has new logistics messages:%s", changedInfo.LogisticsName, progress)
		self.xmppClient.SendChatMessage(changedInfo.Username, messageContent)
	}
}

func (self *PiAssistant) StartService() {
	l4g.Info("Start service!")
	self.updateDownloadStat()
	self.cron.Start()
	l4g.Info("Cron task started!")
	self.chathandler = xmpp.NewChatHandler()
	self.xmppClient.AddHandler(self.chathandler)
	for {
		select {
		case event := <-self.chathandler.GetHandleCh():
			self.handle(event.Stanza.(*xmpp.Message))
		case <-self.stopCh:
			break
		}
	}
}

func (self *PiAssistant) StopService() {
	self.xmppClient.RemoveHandler(self.chathandler)
	self.xmppClient.Disconnect()
	self.cron.Stop()
	self.stopCh <- 1

}

func (self *PiAssistant) handle(message *xmpp.Message) {
	l4g.Info("Receive message from [%s]: %s", message.From, message.Body)
	command := message.Body
	voiceMsgPrefix := "Voice IM:"
	if strings.HasPrefix(command, voiceMsgPrefix) {
		l4g.Debug("Receive voice message: %s", command)
		voiceUrl := strings.TrimSpace(command[len(voiceMsgPrefix):])
		comm, voiceErr := self.convertVoiceToCommand(voiceUrl)
		if voiceErr != nil {
			l4g.Error("Convert voice to command failed: %v", voiceErr)
			self.xmppClient.SendChatMessage(message.From, voiceErr.Error())
			return
		}
		command = comm
	}
	fileMsgPrefix := "I sent you a file through imo:"
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
	l4g.Info("Command from [%s]: %s", message.From, command)
	if command == "help" {
		helpMessage := "\n"
		helpMessage = helpMessage + fmt.Sprintf("%s command:\n%s",
			self.piDownloader.GetServiceName(),
			self.piDownloader.GetComandHelp())
		helpMessage = helpMessage + "------------------\n"
		helpMessage = helpMessage + fmt.Sprintf("%s command:\n%s",
			self.logisticsService.GetServiceName(),
			self.logisticsService.GetComandHelp())
		self.xmppClient.SendChatMessage(message.From, helpMessage)
		return
	}
	username := xmpp.ToBareJID(message.From)
	var resp string
	var err error
	invalidedCommand := false
	switch {
	case self.piDownloader.CheckCommandType(command):
		l4g.Debug("[%s] is download command", command)
		resp, err = self.piDownloader.Process(username, command)
	case self.logisticsService.CheckCommandType(command):
		l4g.Debug("[%s] is logistics command", command)
		resp, err = self.logisticsService.Process(username, command)
	default:
		invalidedCommand = true
	}
	var content string
	if invalidedCommand {
		content = fmt.Sprintf("Invalided command [%s], please type \"help\" for helping information", command)
	} else if err != nil {
		content = err.Error()
	} else {
		content = resp
	}
	self.xmppClient.SendChatMessage(message.From, content)
}

func (self *PiAssistant) convertVoiceToCommand(voiceUrl string) (string, error) {
	text, hasConfidence, convertErr := self.convertVoiceToText(voiceUrl)
	if convertErr != nil {
		return "", convertErr
	}
	if !hasConfidence {
		msg := fmt.Sprintf("Can not understand what you said [%s]!", text)
		return "", errors.New(msg)
	}
	comm := self.convertVoiceTextToCommand(text)
	if comm == "" || len(comm) == 0 {
		errorMsg := "Invalided voice command[" + text + "], please type \"help\" for helping information!"
		return "", errors.New(errorMsg)
	}
	return comm, nil
}

func (self *PiAssistant) convertVoiceTextToCommand(text string) string {
	var command string
	command = self.piDownloader.VoiceToCommand(text)
	if command == "" || len(command) == 0 {
		command = self.logisticsService.VoiceToCommand(text)
	}
	return command
}

func (self *PiAssistant) convertVoiceToText(voiceUrl string) (string, bool, error) {
	text, c, e := speech2text.Speech2Text(voiceUrl)
	if c < self.config.Confidence || e != nil {
		return text, false, e
	}
	return text, true, nil
}
