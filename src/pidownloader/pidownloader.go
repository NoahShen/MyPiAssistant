package pidownloader

import (
	"aria2rpc"
	"bytes"
	l4g "code.google.com/p/log4go"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"utils"
)

type processFunc func(*PiDownloader, []string) (string, error)

type PiDownloader struct {
	torrentDir      string
	commandMap      map[string]processFunc
	commandHelp     map[string]string
	voiceCommandMap map[string]string
}

func NewPidownloader(rpcUrl, torrentDir string) (*PiDownloader, error) {
	piDownloader := new(PiDownloader)
	piDownloader.torrentDir = torrentDir
	aria2rpc.RpcUrl = rpcUrl

	piDownloader.commandMap = map[string]processFunc{
		"add":        (*PiDownloader).addUri,
		"rm":         (*PiDownloader).pause,
		"pause":      (*PiDownloader).pause,
		"pauseall":   (*PiDownloader).pauseAll,
		"unpause":    (*PiDownloader).unpause,
		"unpauseall": (*PiDownloader).unpauseAll,
		"maxspd":     (*PiDownloader).maxspeed,
		"getact":     (*PiDownloader).getActive,
		"getwt":      (*PiDownloader).getWaiting,
		"getstp":     (*PiDownloader).getStopped,
		"addtorrent": (*PiDownloader).addtorrent,
		"getstat":    (*PiDownloader).getAria2GlobalStat,
	}
	piDownloader.voiceCommandMap = map[string]string{
		"全部停止": "pauseall",
		"全部启动": "unpauseall",
		"下载进度": "getact",
		"任务统计": "getstat",
	}

	piDownloader.commandHelp = map[string]string{
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
		"addtorrent": "add bt download task, the torrent must be exist in pi",
		"getstat":    "get download stat",
	}
	return piDownloader, nil
}

func (self *PiDownloader) GetServiceName() string {
	return "PiDownloader"
}

func (self *PiDownloader) VoiceToCommand(voiceText string) string {
	comm := self.voiceCommandMap[voiceText]
	return comm
}

func (self *PiDownloader) CheckCommandType(command string) bool {
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	c := strings.ToLower(comm)
	for commandKey, _ := range self.commandMap {
		if strings.HasPrefix(c, commandKey) {
			l4g.Debug("[%s] is download command", command)
			return true
		}
	}
	return false
}

func (self *PiDownloader) GetComandHelp() string {
	var buffer bytes.Buffer
	for command, helpMsg := range self.commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	buffer.WriteString("voice command:\n")
	for voice, command := range self.voiceCommandMap {
		buffer.WriteString(fmt.Sprintf("[%s] ===> %s\n", voice, command))
	}
	return buffer.String()
}

func (self *PiDownloader) Process(username, command string) (string, error) {
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	f := self.commandMap[comm]
	if f == nil {
		return "", errors.New("Invalided command, please type \"help\" for helping information")
	}
	return f(self, commArr[1:])
}

func (self *PiDownloader) addUri(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	uri := args[0]
	params := make(map[string]string)
	if len(args) > 1 {
		var err error
		params, err = self.parseArgsToMap(args[1:])
		if err != nil {
			return "", err
		}
	}
	l4g.Debug("add Uri params:%v", params)
	gid, err := aria2rpc.AddUri(uri, params)
	if err != nil {
		return "", err
	}
	return "Add successful, gid:" + gid, nil
}

func (self *PiDownloader) parseArgsToMap(args []string) (map[string]string, error) {
	params := make(map[string]string)
	for _, arg := range args {
		argNameValue := strings.SplitN(arg, "=", 2)
		if len(argNameValue) != 2 {
			return nil, errors.New("invalid args!")
		}
		params[argNameValue[0]] = argNameValue[1]
	}
	return params, nil
}

func (self *PiDownloader) addtorrent(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	path := args[0]
	gid, err := aria2rpc.AddTorrent(self.torrentDir + path)
	if err != nil {
		return "", err
	}
	return "Add successful, gid:" + gid, nil
}

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
