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

var helpMessage = `
Command:
add uri            (add download task)

rm gid             (remove specific task by gid)

pause gid          (pause specific task by gid)

pauseall           (pause all tasks)

unpause gid        (unpause specific task by gid)

unpauseall         (unpause all tasks)

maxspd speed       (set max download speed, 0 for unlimit)
 
getact [keys]      (get active tasks)

getwt [keys]       (get waiting tasks)

getstp [keys]     (get stopped tasks)

addtorrent path   (add bt download task)

getstat           (get global stat)
`

type processFunc func(*PiDownloader, []string) (string, error)

//var commandMap = map[string]string{
//	"help":       "c0",
//	"add":        "c1",
//	"rm":         "c2",
//	"pause":      "c3",
//	"pauseall":   "c4",
//	"unpause":    "c5",
//	"unpauseall": "c6",
//	"maxspd":     "c7",
//	"getact":     "c8",
//	"getwt":      "c9",
//	"getstp":     "c10",
//	"addtorrent": "c11",
//	"getstat":    "c87",
//}

type PiDownloader struct {
	torrentDir string
	commandMap map[string]processFunc
}

func NewPidownloader(rpcUrl, torrentDir string) (*PiDownloader, error) {
	piDownloader := new(PiDownloader)
	piDownloader.torrentDir = torrentDir
	aria2rpc.RpcUrl = rpcUrl

	piDownloader.commandMap = map[string]processFunc{
		"help":       (*PiDownloader).help,
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
	return piDownloader, nil
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

func (self *PiDownloader) Process(username, command string) (string, error) {
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	f := self.commandMap[comm]
	if f == nil {
		return "", errors.New("Invalided command, please type \"help\" for helping information")
	}
	return f(self, commArr[1:])
}

func (self *PiDownloader) help(args []string) (string, error) {
	return helpMessage, nil
}

func (self *PiDownloader) addUri(args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("missing args!")
	}
	uri := args[0]
	gid, err := aria2rpc.AddUri(uri, nil)
	if err != nil {
		return "", err
	}
	return "Add successful, gid:" + gid, nil
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
