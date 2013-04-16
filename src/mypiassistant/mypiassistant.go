package main

import (
	"code.google.com/p/goconf/conf"
	l4g "code.google.com/p/log4go"
	"flag"
	"github.com/robfig/cron"
	"pidownloader"
	"runtime"
	"speech2text"
	"strings"
	"xmpp"
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
	UpdateCron string
	XmppHost   string
	XmppUser   string
	XmppPwd    string
	RpcUrl     string
	RpcVersion string
	TorrentDir string
	Confidence float64
}

type PiAssistant struct {
	piDownloader *pidownloader.PiDownloader
	xmppClient   *xmpp.XmppClient
	config       *config
	stopCh       chan int
	chathandler  xmpp.Handler
	cron         *cron.Cron
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
	if config.UpdateCron, err = c.GetString("aria2", "update_cron"); err != nil {
		return nil, err
	}
	if config.TorrentDir, err = c.GetString("aria2", "torrent_dir"); err != nil {
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
	piDer, err := pidownloader.NewPidownloader(config.RpcUrl, config.TorrentDir)
	if err != nil {
		l4g.Error("Create PiDownloader error: %v", err)
		return nil, err
	}

	pi := &PiAssistant{}
	pi.config = config
	pi.piDownloader = piDer
	pi.xmppClient = xmpp.NewXmppClient()
	pi.cron = cron.New()
	pi.cron.AddFunc(pi.config.UpdateCron, func() {
		pi.updateDownloadStat()
	})
	return pi, nil
}

func (self *PiAssistant) Init() error {
	_, statErr := self.piDownloader.ProcessCommandNo("87", nil)
	if statErr != nil {
		l4g.Error("Call aria2 error: %v", statErr)
		return statErr
	}
	l4g.Info("Call aria2 successful!")
	// connect xmpp server
	xmppErr := self.xmppClient.Connect(self.config.XmppHost, self.config.XmppUser, self.config.XmppPwd)
	if xmppErr != nil {
		l4g.Error("Connect xmpp server error: %v", xmppErr)
		return xmppErr
	}
	l4g.Info("Xmpp is connected!")
	self.cron.Start()
	return nil
}

func (self *PiAssistant) updateDownloadStat() {
	l4g.Debug("updateDownloadStat!")
	status, statErr := self.piDownloader.ProcessCommandNo("87", nil)
	if statErr != nil {
		self.xmppClient.Send(statErr.Error())
	} else {
		self.xmppClient.Send(status)
	}
}
func (self *PiAssistant) StartService() {
	l4g.Info("Start service!")
	self.updateDownloadStat()
	self.chathandler = xmpp.NewChatHandler()
	self.xmppClient.AddHandler(self.chathandler)
	for {
		select {
		case msg := <-self.chathandler.GetHandleCh():
			self.handle(msg.(xmpp.Chat))
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

var voiceMap = map[string]string{
	"帮助":   "c0",
	"全部停止": "c4",
	"全部启动": "c6",
	"下载进度": "c8",
	"任务统计": "c87",
}

func (self *PiAssistant) handle(chatMessage xmpp.Chat) {
	command := chatMessage.Text
	if strings.HasPrefix(command, "Voice IM:") {
		l4g.Debug("Receive voice message!")
		voiceUrl := strings.TrimSpace(command[len("Voice IM:"):])
		text, understand, convertErr := self.convertVoiceToText(voiceUrl)
		if convertErr != nil {
			l4g.Error("Convert voice to text failed: %s", convertErr.Error())
			replyChat := &xmpp.Chat{chatMessage.Remote, chatMessage.Type, convertErr.Error()}
			self.xmppClient.Send(replyChat)
			return
		}
		if !understand {
			msg := "Can not hear what you're saying! Please try again."
			replyChat := &xmpp.Chat{chatMessage.Remote, chatMessage.Type, msg}
			self.xmppClient.Send(replyChat)
			return
		}
		comm := voiceMap[text]
		if comm == "" || len(comm) == 0 {
			errorMsg := "Can not understand your command[" + text + "]!"
			replyChat := &xmpp.Chat{chatMessage.Remote, chatMessage.Type, errorMsg}
			self.xmppClient.Send(replyChat)
			return
		}
		l4g.Debug("voice text ==> command :%s ===> %s", text, comm)
		command = comm
	}
	resp, err := self.piDownloader.Process(command)
	var replyChat *xmpp.Chat
	if err != nil {
		replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, err.Error()}
	} else {
		replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, resp}
	}
	self.xmppClient.Send(replyChat)
}

func (self *PiAssistant) convertVoiceToText(voiceUrl string) (string, bool, error) {
	text, c, e := speech2text.Speech2Text(voiceUrl)
	if c < self.config.Confidence || e != nil {
		return "", false, e
	}
	return text, true, nil
}
