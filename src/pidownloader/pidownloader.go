package pidownloader

import (
	"aria2rpc"
	"bytes"
	"code.google.com/p/goconf/conf"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"utils"
	"xmpp"
)

var helpMessage = `
Command:
addUri[c1] uri

remove[c2] gid

pause[c3] gid;

pauseall[c4]

Unpause[c5] gid

unpauseall[c6]

GetStatus[c7] gid [keys]
 
getactive[c8] [keys]

GetWaiting[c9] [keys]

GetStopped[c10] [keys]

getstat[c87]

MaxSpeed[c11] speed
`

var commandMap = map[string]string{
	"help":       "c0",
	"pauseall":   "c4",
	"unpauseall": "c6",
	"getactive":  "c8",
	"getstat":    "c87",
}

type config struct {
	UpdateInterval int
	XmppHost       string
	XmppUser       string
	XmppPwd        string
	RpcUrl         string
	RpcVersion     string
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
	if config.UpdateInterval, err = c.GetInt("aria2", "update_interval"); err != nil {
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

	aria2rpc.RpcUrl = self.config.RpcUrl
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
		case <-time.After((time.Duration)(self.config.UpdateInterval) * time.Second):
			status, statErr := self.getAria2GlobalStat()
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

func (self *PiDownloader) StopService() {
	self.stopCh <- 1
}

func (self *PiDownloader) process(command string) (string, error) {
	log.Println("receive command:", command)
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	if strings.HasPrefix(comm, "c") {
		return self.processCommandNo(comm[1:], commArr[1:])
	} else {
		c := strings.ToLower(comm)
		commandNo := commandMap[c]
		log.Println("mapped command no:", commandNo)
		if commandNo != "" && len(commandNo) > 0 {
			return self.processCommandNo(commandNo[1:], commArr[1:])
		} else {
			return "", errors.New("Error command, please type \"help\" for helping information")
		}
	}
	return "OK", nil
}

func (self *PiDownloader) processCommandNo(number string, args []string) (string, error) {
	cNumber, numErr := strconv.Atoi(number)
	if numErr != nil {
		return "", errors.New("Number Command must be \"c\" + number")
	}
	switch cNumber {
	case 0:
		return helpMessage, nil
	case 4:
		return self.pauseAll()
	case 6:
		return self.unpauseAll()
	case 8:
		return self.getActive(args)
	case 87:
		return self.getAria2GlobalStat()
	default:
		return "", errors.New("Error command no, please type \"help\" for helping information")
	}
	return "", nil
}

func (self *PiDownloader) getActive(args []string) (string, error) {
	var keys []string
	if args == nil || len(args) == 0 {
		keys = []string{"gid", "totalLength", "completedLength", "downloadSpeed"}
	} else {
		keys = args
	}
	actives, err := aria2rpc.GetActive(keys)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	for _, task := range actives {
		gid := task["gid"].(string)
		speed := utils.FormatSizeString(task["downloadSpeed"].(string))
		completed, _ := strconv.ParseFloat(task["completedLength"].(string), 64)
		total, _ := strconv.ParseFloat(task["totalLength"].(string), 64)
		buffer.WriteString(fmt.Sprintf("gid: %s;downloadSpeed: %s;progress: %.2f%%\n", gid, speed, completed*100/total))
	}

	return buffer.String(), nil
}

func (self *PiDownloader) pauseAll() (string, error) {
	_, err := aria2rpc.PauseAll()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) unpauseAll() (string, error) {
	_, err := aria2rpc.UnpauseAll()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) getAria2GlobalStat() (string, error) {
	globalStat, err := aria2rpc.GetGlobalStat()
	if err != nil {
		log.Println("GetGlobalStat error:", err)
		return "", nil
	}
	speed := utils.FormatSizeString(globalStat["downloadSpeed"].(string))
	numActive := globalStat["numActive"].(string)
	numStopped := globalStat["numStopped"].(string)
	numWaiting := globalStat["numWaiting"].(string)
	return "spd:" + speed + ";" +
		"act:" + numActive + ";" +
		"wait:" + numWaiting + ";" +
		"stop:" + numStopped, nil
}
