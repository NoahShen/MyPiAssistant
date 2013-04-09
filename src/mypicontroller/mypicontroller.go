package main

import (
	"code.google.com/p/goconf/conf"
	"flag"
	"fmt"
	"pidownloader"
	"time"
	"xmpp"
)

var (
	configpath = flag.String("configpath", "", "path of config file")
)

func main() {
	flag.Parse()
	piController, err := NewPiController(*configpath) //"../config/pidownloader.conf"
	if err != nil {
		fmt.Println("start error:", err)
		return
	}

	if initErr := piController.Init(); initErr != nil {
		fmt.Println("init error:", initErr)
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
}

type PiController struct {
	piDownloader *pidownloader.PiDownloader
	xmppClient   *xmpp.XmppClient
	config       *config
	stopCh       chan int
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
	config, configErr := loadConfig(configPath)
	if configErr != nil {
		return nil, configErr
	}
	piDer, err := pidownloader.NewPidownloader(config.RpcUrl, config.TorrentDir)
	if err != nil {
		return nil, err
	}

	pi := &PiController{}
	pi.config = config
	pi.piDownloader = piDer
	pi.xmppClient = xmpp.NewXmppClient()
	return pi, nil
}

func (self *PiController) Init() error {
	//make sure aria2 is running
	_, statErr := self.piDownloader.Process("getstat")
	if statErr != nil {
		return statErr
	}
	// connect xmpp server
	xmppErr := self.xmppClient.Connect(self.config.XmppHost, self.config.XmppUser, self.config.XmppPwd)
	if xmppErr != nil {
		return xmppErr
	}
	return nil
}

func (self *PiController) StartService() {

	chathandler := xmpp.NewChatHandler()
	self.xmppClient.AddHandler(chathandler)
	for {
		select {
		case msg := <-chathandler.GetHandleCh():
			self.handle(msg.(xmpp.Chat))
		case <-time.After((time.Duration)(self.config.UpdateInterval) * time.Second):
			status, statErr := self.piDownloader.ProcessCommandNo("87", nil)
			if statErr != nil {
				self.xmppClient.Send(statErr.Error())
			} else {
				self.xmppClient.Send(status)
			}

		case <-self.stopCh:
			break
		}
	}
}

func (self *PiController) StopService() {
	self.xmppClient.Disconnect()
	self.stopCh <- 1

}

func (self *PiController) handle(chatMessage xmpp.Chat) {
	command := chatMessage.Text
	resp, err := self.piDownloader.Process(command)
	var replyChat *xmpp.Chat
	if err != nil {
		replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, "error:" + err.Error()}
	} else {
		replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, resp}
	}
	self.xmppClient.Send(replyChat)
}
