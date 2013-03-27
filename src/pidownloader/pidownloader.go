package pidownloader

import (
	"aria2rpc"
	"code.google.com/p/goconf/conf"
	"errors"
	"log"
	"strconv"
	"strings"
	"xmpp"
)

var helpMessage = `
Command:
addUri[c1] uri

remove[c2] gid

pause[c3] gid;

PauseAll[c4]

Unpause[c5] gid

UnpauseAll[c6]

GetStatus[c7] gid [keys]
 
GetActive[c8] [keys]

GetWaiting[c9] [keys]

GetStopped[c10] [keys]

GetGlobalStat[c87]

MaxSpeed[c11] speed
`

var commandMap = map[string]string{
	"help": "c0",
}

type config struct {
	XmppHost   string
	XmppUser   string
	XmppPwd    string
	RpcUrl     string
	RpcVersion string
}

type PiDownloader struct {
	xmppClient *xmpp.XmppClient
	config     *config
	stopCh     chan int
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

	return config, nil
}

func NewPidownloader(configPath string) (*PiDownloader, error) {

	config, configErr := loadConfig(configPath)
	if configErr != nil {
		return nil, configErr
	}
	piDownloader := new(PiDownloader)
	piDownloader.config = config

	piDownloader.xmppClient = xmpp.NewXmppClient()
	piDownloader.stopCh = make(chan int, 1)
	return piDownloader, nil
}

func (self *PiDownloader) Init() error {
	//make sure aria2 is running
	_, aria2Error := aria2rpc.GetGlobalStat()
	if aria2Error != nil {
		return aria2Error
	}

	// connect xmpp server
	xmppErr := self.xmppClient.Connect(self.config.XmppHost, self.config.XmppUser, self.config.XmppPwd)
	if xmppErr != nil {
		return xmppErr
	}
	return nil
}

func (self *PiDownloader) StartService() {
	chathandler := xmpp.NewChatHandler()
	self.xmppClient.AddHandler(chathandler)
	for {
		select {
		case msg := <-chathandler.GetHandleCh():
			chatMessage := msg.(xmpp.Chat)
			command := chatMessage.Text
			resp, err := self.process(command)
			var replyChat *xmpp.Chat
			if err != nil {
				replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, "error:" + err.Error()}
			} else {
				replyChat = &xmpp.Chat{chatMessage.Remote, chatMessage.Type, resp}
			}
			self.xmppClient.Send(replyChat)
		case <-self.stopCh:
			break
		}
	}
}

func (self *PiDownloader) StopService() {
	self.stopCh <- 1
}

func (self *PiDownloader) process(command string) (string, error) {
	log.Println("receive command:", command)
	if strings.HasPrefix(command, "c") {
		return self.processCommandNo(command[1:])
	} else {
		c := strings.ToLower(command)
		commandNo := commandMap[c]
		log.Println("mapped command no:", commandNo)
		if commandNo != "" && len(commandNo) > 0 {
			return self.processCommandNo(commandNo[1:])
		} else {
			return "", errors.New("Error command, please type \"help\" for helping information")
		}
	}
	return "OK", nil
}

func (self *PiDownloader) processCommandNo(number string) (string, error) {
	cNumber, numErr := strconv.Atoi(number)
	if numErr != nil {
		return "", errors.New("Number Command must be \"c\" + number")
	}
	switch cNumber {
	case 0:
		return helpMessage, nil
	}
	return "", nil
}
