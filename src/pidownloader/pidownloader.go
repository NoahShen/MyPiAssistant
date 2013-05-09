package pidownloader

import (
	"bytes"
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NoahShen/aria2rpc"
	"github.com/robfig/cron"
	"path"
	"service"
	"strconv"
	"strings"
	"utils"
)

type processFunc func(*PiDownloader, []string) (string, error)

var commandHelp = map[string]string{
	"add":        "add download url",
	"rm":         "remove specific task by gid",
	"pause":      "get current logistics message, like getlogi name or getlogi company logistics id",
	"pauseall":   "pause all tasks",
	"unpause":    "unpause specific task by gid",
	"unpauseall": "unpause all tasks",
	"maxspd":     "set max download speed, 0 for unlimit",
	"getact":     "get active tasks",
	"getwt":      "get waiting tasks",
	"getstp":     "get stopped tasks",
	"getstat":    "get download stat",
}

type PiDownloader struct {
	commandMap      map[string]processFunc
	voiceCommandMap map[string]string
	pushMsgChannel  chan<- *service.PushMessage
	cron            *cron.Cron
}

type config struct {
	RpcUrl         string `json:"rpcUrl,omitempty"`
	RpcVersion     string `json:"rpcVersion,omitempty"`
	StatUpdateCron string `json:"statUpdateCron,omitempty"`
}

func (self *PiDownloader) GetServiceName() string {
	return "pidownloader"
}

func (self *PiDownloader) Init(configRawMsg *json.RawMessage, pushCh chan<- *service.PushMessage) error {
	var c config
	err := json.Unmarshal(*configRawMsg, &c)
	if err != nil {
		return err
	}
	aria2rpc.RpcUrl = c.RpcUrl
	aria2rpc.RpcVersion = c.RpcVersion
	self.pushMsgChannel = pushCh
	self.cron = cron.New()
	self.cron.AddFunc(c.StatUpdateCron, func() {
		self.updateDownloadStat()
	})
	self.commandMap = map[string]processFunc{
		"add":        (*PiDownloader).addUri,
		"rm":         (*PiDownloader).remove,
		"pause":      (*PiDownloader).pause,
		"pauseall":   (*PiDownloader).pauseAll,
		"unpause":    (*PiDownloader).unpause,
		"unpauseall": (*PiDownloader).unpauseAll,
		"maxspd":     (*PiDownloader).maxspeed,
		"getact":     (*PiDownloader).getActive,
		"getwt":      (*PiDownloader).getWaiting,
		"getstp":     (*PiDownloader).getStopped,
		"getstat":    (*PiDownloader).getAria2GlobalStat,
		"file":       (*PiDownloader).handleFile,
	}
	self.voiceCommandMap = map[string]string{
		"全部停止": "pauseall",
		"全部启动": "unpauseall",
		"下载进度": "getact",
		"任务统计": "getstat",
	}

	_, statErr := self.Handle("", "getstat", nil)
	if statErr != nil {
		l4g.Error("Call aria2 error: %v", statErr)
		return statErr
	}
	return nil
}

func (self *PiDownloader) StartService() error {
	self.cron.Start()
	return nil
}

func (self *PiDownloader) Stop() error {
	self.cron.Stop()
	return nil
}

func (self *PiDownloader) CommandFilter(command string, args []string) bool {
	if _, ok := self.voiceCommandMap[command]; ok {
		return true
	}

	if _, ok := self.commandMap[command]; ok {
		if command == "file" {
			fileUrl := args[0]
			i := strings.LastIndex(fileUrl, "/")
			fileName := fileUrl[i+1:]
			return fileName == "aria2.down"
		}
		return true
	}
	return false
}

func (self *PiDownloader) GetHelpMessage() string {
	var buffer bytes.Buffer
	for command, helpMsg := range commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	buffer.WriteString("voice command:\n")
	for voice, command := range self.voiceCommandMap {
		buffer.WriteString(fmt.Sprintf("[%s] ===> %s\n", voice, command))
	}
	return buffer.String()
}

func (self *PiDownloader) updateDownloadStat() {
	status, statErr := self.Handle("", "getstat", nil)
	pushMsg := &service.PushMessage{}
	pushMsg.Type = service.Status
	if statErr != nil {
		l4g.Error("Get download stat error: %v", statErr)
		pushMsg.Message = statErr.Error()
	} else {
		pushMsg.Message = status
	}
	self.pushMsgChannel <- pushMsg
}

func (self *PiDownloader) Handle(username, command string, args []string) (string, error) {
	comm := self.voiceCommandMap[command]
	if comm == "" || len(comm) == 0 {
		comm = command
	}

	f := self.commandMap[comm]
	if f == nil {
		return "", errors.New("Invalided download command, please type \"help\" for helping information")
	}
	return f(self, args)
}

func (self *PiDownloader) handleFile(args []string) (string, error) {
	filePath := args[1]
	l4g.Debug("Starting parse commandfile: %s", filePath)
	commands, err := self.parseCommandFile(filePath)
	if err != nil {
		return "", err
	}
	for _, command := range commands {
		l4g.Debug("Exec command from file: %s", command)
		commArr := strings.Split(command, " ")
		comm := commArr[0]
		c := strings.ToLower(comm)
		_, execErr := self.Handle("", c, commArr[1:])
		if execErr != nil {
			return "", execErr
		}
	}
	return fmt.Sprintf("Command file executed successful! total: %d commands.", len(commands)), nil
}

func (self *PiDownloader) addUri(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	uris := make([]string, 0)
	params := make(map[string]interface{})
	for _, arg := range args {
		if utils.IsHttpUrl(arg) {
			uris = append(uris, arg)
		} else {
			argNameValue := strings.SplitN(arg, "=", 2)
			if len(argNameValue) != 2 {
				return "", errors.New("invalid args!")
			}
			values := strings.Split(strings.TrimSpace(argNameValue[1]), ";")
			if len(values) == 1 {
				params[argNameValue[0]] = values[0]
			} else {
				params[argNameValue[0]] = values
			}

		}

	}
	gids := make([]string, 0)
	for _, uri := range uris {
		l4g.Debug("Dowanload uri: %v", uri)
		l4g.Debug("Dowanload params: %v", params)
		gid, err := aria2rpc.AddUri(uri, params)
		if err != nil {
			return "", err
		}
		gids = append(gids, gid)
	}

	return fmt.Sprintf("Add successful, gids:%v", gids), nil
}

//func (self *PiDownloader) addGdriveId(params map[string]interface{}) map[string]interface{} {
//	headers := params["header"]
//	if headers == nil {
//		headers = []string{"Cookie:gdriveid=" + self.gdriveid}
//		params["header"] = headers
//	} else {

//		switch headers.(type) {
//		case []string:
//			containGdrive := false
//			headerArr := headers.([]string)
//			for _, header := range headerArr {
//				if strings.Contains(header, "gdriveid") {
//					containGdrive = true
//				}
//			}
//			if !containGdrive {
//				headerArr = append(headerArr, "Cookie:gdriveid="+self.gdriveid)
//			}
//		case string:
//			headerStr := headers.(string)
//			if !strings.Contains(headerStr, "gdriveid") {
//				headers := []string{headerStr, "Cookie:gdriveid=" + self.gdriveid}
//				params["header"] = headers
//			}
//		}

//	}
//	return params
//}

//func (self *PiDownloader) addtorrent(args []string) (string, error) {
//	if args == nil || len(args) == 0 {
//		return "", errors.New("missing args!")
//	}
//	path := args[0]
//	gid, err := aria2rpc.AddTorrent(self.torrentDir + path)
//	if err != nil {
//		return "", err
//	}
//	return "Add successful, gid:" + gid, nil
//}

func (self *PiDownloader) remove(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	gid := args[0]
	rgid, err := aria2rpc.Remove(gid, true)
	if err != nil {
		return "", err
	}
	return "Remove successful, gid:" + rgid, nil
}

func (self *PiDownloader) pause(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	gid := args[0]
	_, err := aria2rpc.Pause(gid, true)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) unpause(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	gid := args[0]
	_, err := aria2rpc.Unpause(gid)
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) maxspeed(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	params := make(map[string]string)
	params["max-overall-download-limit"] = args[0]
	_, err := aria2rpc.ChangeGlobalOption(params)
	if err != nil {
		return "", nil
	}
	return "OK", nil
}

func (self *PiDownloader) getActive(args []string) (string, error) {
	keys := []string{"gid", "totalLength", "completedLength", "downloadSpeed", "bittorrent", "files"}
	tasks, err := aria2rpc.GetActive(keys)
	if err != nil {
		return "", err
	}
	return self.formatOutput(tasks)
}

func (self *PiDownloader) getWaiting(args []string) (string, error) {
	keys := []string{"gid", "totalLength", "completedLength", "bittorrent", "files"}
	tasks, err := aria2rpc.GetWaiting(0, 100, keys)
	if err != nil {
		return "", err
	}
	return self.formatOutput(tasks)
}

func (self *PiDownloader) getStopped(args []string) (string, error) {
	keys := []string{"gid", "totalLength", "completedLength", "bittorrent", "files", "status", "errorCode"}
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
		l4g.Debug("=================\n%s", task)
		gid := task["gid"].(string)
		buffer.WriteString(fmt.Sprintf("gid: %s\n", gid))

		title := self.getTitle(task)
		buffer.WriteString(fmt.Sprintf("title: %s\n", title))

		dSpd := task["downloadSpeed"]
		if dSpd != nil {
			speed := utils.FormatSizeString(dSpd.(string))
			buffer.WriteString(fmt.Sprintf("spd: %s\n", speed))
		}

		total, _ := strconv.ParseFloat(task["totalLength"].(string), 64)
		totalFmt := utils.FormatSize(int64(total))
		buffer.WriteString(fmt.Sprintf("total: %s\n", totalFmt))

		completed, _ := strconv.ParseFloat(task["completedLength"].(string), 64)
		buffer.WriteString(fmt.Sprintf("prog: %.2f%%\n", completed*100/total))

		if dSpd != nil {
			spd, _ := strconv.Atoi(dSpd.(string))
			var timeLeftFmt string
			if spd == 0 {
				timeLeftFmt = "N/A"
			} else {
				timeLeft := int64(total-completed) / int64(spd)
				timeLeftFmt = utils.FormatTime(timeLeft)
			}
			buffer.WriteString(fmt.Sprintf("tiemleft: %s\n", timeLeftFmt))
		}

		status := task["status"]
		if status != nil {
			buffer.WriteString(fmt.Sprintf("status: %s\n", status.(string)))
		}

		errorCode := task["errorCode"]
		if status != nil {
			buffer.WriteString(fmt.Sprintf("statusCode: %s\n", errorCode.(string)))
		}
		buffer.WriteString("==================\n")
	}
	return buffer.String(), nil
}

func (self *PiDownloader) getTitle(task map[string]interface{}) string {
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

func (self *PiDownloader) pauseAll(args []string) (string, error) {
	_, err := aria2rpc.PauseAll()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) unpauseAll(args []string) (string, error) {
	_, err := aria2rpc.UnpauseAll()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *PiDownloader) getAria2GlobalStat(args []string) (string, error) {
	globalStat, err := aria2rpc.GetGlobalStat()
	if err != nil {
		return "", err
	}
	speed := utils.FormatSizeString(globalStat["downloadSpeed"].(string))
	numActive := globalStat["numActive"].(string)
	numStopped := globalStat["numStopped"].(string)
	numWaiting := globalStat["numWaiting"].(string)
	return fmt.Sprintf("spd:%s ; act:%s ; wait:%s ; stop:%s", speed, numActive, numWaiting, numStopped), nil
}
