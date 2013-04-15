package main

import (
	"code.google.com/p/goconf/conf"
	l4g "code.google.com/p/log4go"
	"flag"
	"pidownloader"
	"runtime"
	"speech2text"
	"strings"
	"time"
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
	piController, err := NewPiController(*configPath) //"../config/pidownloader.conf"
	if err != nil {
		return
	}

	if initErr := piController.Init(); initErr != nil {
		return
	}
	piController.StartService()
}

type config struct {
	UpdateInterval int
	XmppHost       string
	XmppUser       string
	XmppPwd        string
	RpcUrl         string
	RpcVersion     string
	TorrentDir     string
	Confidence     float64
}

type PiController struct {
	piDownloader *pidownloader.PiDownloader
	xmppClient   *xmpp.XmppClient
	config       *config
	stopCh       chan int
	chathandler  xmpp.Handler
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
	if config.UpdateInterval, err = c.GetInt("aria2", "update_interval"); err != nil {
		return nil, err
	}
	if config.TorrentDir, err = c.GetString("aria2", "torrent_dir"); err != nil {
		return nil, err
	}

	return config, nil
}

func NewPiController(configPath string) (*PiController, error) {
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

	pi := &PiController{}
	pi.config = config
	pi.piDownloader = piDer
	pi.xmppClient = xmpp.NewXmppClient()
	return pi, nil
}

func (self *PiController) Init() error {
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
	return nil
}

func (self *PiController) StartService() {
	l4g.Info("Start service!")
	state, statErr := self.piDownloader.ProcessCommandNo("87", nil)
	if statErr != nil {
		self.xmppClient.Send(statErr.Error())
	} else {
		self.xmppClient.Send(state)
	}
	self.chathandler = xmpp.NewChatHandler()
	self.xmppClient.AddHandler(self.chathandler)

	lastUpdate := time.Now()
	for {
		// calculate next update status time
		escaped := time.Since(lastUpdate)
		updateAfter := time.Duration(self.config.UpdateInterval)*time.Second - escaped
		if updateAfter < 0 {
			updateAfter = 0
		}
		l4g.Debug("updateAfter: %v", updateAfter)
		select {
		case msg := <-self.chathandler.GetHandleCh():
			self.handle(msg.(xmpp.Chat))
		case <-time.After(updateAfter):
			status, statErr := self.piDownloader.ProcessCommandNo("87", nil)
			if statErr != nil {
				self.xmppClient.Send(statErr.Error())
			} else {
				self.xmppClient.Send(status)
			}
			lastUpdate = time.Now()
		case <-self.stopCh:
			break
		}
	}
}

func (self *PiController) StopService() {
	self.xmppClient.RemoveHandler(self.chathandler)
	self.xmppClient.Disconnect()
	self.stopCh <- 1

}

var voiceMap = map[string]string{
	"帮助":   "c0",
	"全部停止": "c4",
	"全部启动": "c6",
	"下载进度": "c8",
	"任务统计": "c87",
}

func (self *PiController) handle(chatMessage xmpp.Chat) {
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

func (self *PiController) convertVoiceToText(voiceUrl string) (string, bool, error) {
	text, c, e := speech2text.Speech2Text(voiceUrl)
	if c < self.config.Confidence || e != nil {
		return "", false, e
	}
	return text, true, nil
}
