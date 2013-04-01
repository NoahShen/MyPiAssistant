package pidownloader

import (
	"aria2rpc"
	"bytes"
	"code.google.com/p/goconf/conf"
	"errors"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"time"
	"utils"
	"xmpp"
)

var helpMessage = `
Command:
The string in [] is command's alias.For example: addUri http..  equals c1 http...

add[c1] uri         (add download task)

rm[c2] gid             (remove specific task by gid)

pause[c3] gid          (pause specific task by gid)

pauseall[c4]           (pause all tasks)

unpause[c5] gid        (unpause specific task by gid)

unpauseall[c6]         (unpause all tasks)

maxspd[c7] speed       (set max download speed, 0 for unlimit)
 
getact[c8] [keys]      (get active tasks)

getwt[c9] [keys]       (get waiting tasks)

getstp[c10] [keys]     (get stopped tasks)

addtorrent[c11] path   (add bt download task)

getstat[c87]           (get global stat)
`

var commandMap = map[string]string{
	"help":       "c0",
	"add":        "c1",
	"rm":         "c2",
	"pause":      "c3",
	"pauseall":   "c4",
	"unpause":    "c5",
	"unpauseall": "c6",
	"maxspd":     "c7",
	"getact":     "c8",
	"getwt":      "c9",
	"getstp":     "c10",
	"addtorrent": "c11",
	"getstat":    "c87",
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
	if config.TorrentDir, err = c.GetString("aria2", "torrent_dir"); err != nil {
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
	case 1:
		return self.addUri(args)
	case 2:
		return self.remove(args)
	case 3:
		return self.pause(args)
	case 4:
		return self.pauseAll()
	case 5:
		return self.unpause(args)
	case 6:
		return self.unpauseAll()
	case 7:
		return self.maxspeed(args)
	case 8:
		return self.getActive(args)
	case 9:
		return self.getWaiting(args)
	case 10:
		return self.getStopped(args)
	case 11:
		return self.addtorrent(args)
	case 87:
		return self.getAria2GlobalStat()
	default:
		return "", errors.New("Error command no, please type \"help\" for helping information")
	}
	return "", nil
}

func (self *PiDownloader) addUri(args []string) (string, error) {
	uri := args[0]
	gid, err := aria2rpc.AddUri(uri, nil)
	if err != nil {
		return "", err
	}
	return "Add successful, gid:" + gid, nil
}

func (self *PiDownloader) addtorrent(args []string) (string, error) {
	path := args[0]
	gid, err := aria2rpc.AddTorrent(self.config.TorrentDir + path)
	if err != nil {
		return "", err
	}
	return "Add successful, gid:" + gid, nil
}

func (self *PiDownloader) remove(args []string) (string, error) {
	gid := args[0]
	rgid, err := aria2rpc.Remove(gid, true)
	if err != nil {
		return "", err
	}
	log.Println(gid)
	return "Remove successful, gid:" + rgid, nil
}

func (self *PiDownloader) pause(args []string) (string, error) {
	gid := args[0]
	_, err := aria2rpc.Pause(gid, true)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) unpause(args []string) (string, error) {
	gid := args[0]
	_, err := aria2rpc.Unpause(gid)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) maxspeed(args []string) (string, error) {
	params := make(map[string]string)
	params["max-overall-download-limit"] = args[0]
	_, err := aria2rpc.ChangeGlobalOption(params)
	if err != nil {
		return "", nil
	}
	return "OK", nil
}

func (self *PiDownloader) getActive(args []string) (string, error) {
	//var keys []string
	//if args == nil || len(args) == 0 {
	//	keys = []string{"gid", "totalLength", "completedLength", "downloadSpeed"}
	//} else {
	//	keys = args
	//}
	keys := []string{"gid", "totalLength", "completedLength", "downloadSpeed", "bittorrent", "files"}
	tasks, err := aria2rpc.GetActive(keys)
	if err != nil {
		return "", err
	}
	return self.formatOutput(tasks)
}

func (self *PiDownloader) getWaiting(args []string) (string, error) {
	keys := []string{"gid", "totalLength", "completedLength", "downloadSpeed", "bittorrent", "files"}
	tasks, err := aria2rpc.GetWaiting(0, 100, keys)
	if err != nil {
		return "", err
	}
	return self.formatOutput(tasks)
}

func (self *PiDownloader) getStopped(args []string) (string, error) {
	keys := []string{"gid", "totalLength", "completedLength", "downloadSpeed", "bittorrent", "files"}
	tasks, err := aria2rpc.GetStopped(0, 100, keys)
	if err != nil {
		return "", err
	}
	return self.formatOutput(tasks)
}

func (self *PiDownloader) formatOutput(tasks []map[string]interface{}) (string, error) {
	if tasks == nil || len(tasks) == 0 {
		return "no records", nil
	}
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for _, task := range tasks {
		gid := task["gid"].(string)
		speed := utils.FormatSizeString(task["downloadSpeed"].(string))
		completed, _ := strconv.ParseFloat(task["completedLength"].(string), 64)
		total, _ := strconv.ParseFloat(task["totalLength"].(string), 64)
		title := self.getTitle(task)
		buffer.WriteString(fmt.Sprintf("gid: %s\ntitle: %s\nspd: %s\nprog: %.2f%%\n", gid, title, speed, completed*100/total))
		buffer.WriteString("==================\n")
	}
	return buffer.String(), nil
}

func (self *PiDownloader) getTitle(task map[string]interface{}) string {
	log.Println(task)
	// get bt task title
	bt := task["bittorrent"]
	if bt != nil {
		info := (bt.(map[string]interface{}))["info"]
		if info != nil {
			name := (info.(map[string]interface{}))["name"]
			if name != nil {
				return name.(string)
			}
		}
	}
	// http task title
	files := task["files"]
	if files != nil {
		file := files.([]interface{})[0].(map[string]interface{})
		filePath := file["path"]
		if filePath != nil {
			return path.Base(filePath.(string))
		}
	}
	return "No title"
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
	return fmt.Sprintf("spd:%s ; act:%s ; wait:%s ; stop:%s", speed, numActive, numWaiting, numStopped), nil
}
