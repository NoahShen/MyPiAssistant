package pidownloader

import (
	"aria2rpc"
	"bytes"
	"errors"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"utils"
)

var helpMessage = `
Command:
The string in [] is command's alias.For example: addUri http..  equals c1 http...

add[c1] uri            (add download task)

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

type PiDownloader struct {
	torrentDir string
}

func NewPidownloader(rpcUrl, torrentDir string) (*PiDownloader, error) {
	piDownloader := new(PiDownloader)
	piDownloader.torrentDir = torrentDir
	aria2rpc.RpcUrl = rpcUrl
	return piDownloader, nil
}

func (self *PiDownloader) Process(command string) (string, error) {
	log.Println("receive command:", command)
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	if strings.HasPrefix(comm, "c") {
		return self.ProcessCommandNo(comm[1:], commArr[1:])
	} else {
		c := strings.ToLower(comm)
		commandNo := commandMap[c]
		log.Println("mapped command no:", commandNo)
		if commandNo != "" && len(commandNo) > 0 {
			return self.ProcessCommandNo(commandNo[1:], commArr[1:])
		} else {
			return "", errors.New("The command[" + command + "] is invalid, please type \"help\" for helping information")
		}
	}
	return "OK", nil
}

func (self *PiDownloader) ProcessCommandNo(number string, args []string) (string, error) {
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
