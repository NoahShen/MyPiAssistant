package pidownloader

import (
	"bufio"
	l4g "code.google.com/p/log4go"
	"fmt"
	"io"
	"os"
	"strings"
	"utils"
)

func (self *PiDownloader) parseCommandFile(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	commands := make([]string, 0)
	var lastCommand string
	for {
		line, e := readln(reader)
		if e != nil {
			if e == io.EOF {
				if lastCommand != "" && len(lastCommand) > 0 {
					commands = append(commands, lastCommand)
				}
				break
			} else {
				return nil, e
			}
		}
		l4g.Debug("Read line from %s: %s", filePath, line)
		trimLine := strings.TrimSpace(line)
		if trimLine == "" || len(trimLine) == 0 {
			if lastCommand != "" && len(lastCommand) > 0 {
				commands = append(commands, lastCommand)
			}
			lastCommand = ""
			continue
		}
		if utils.IsHttpUrl(trimLine) {
			lastCommand = "add " + trimLine
		} else {
			paramNameValue := strings.SplitN(trimLine, "=", 2)
			lastCommand = lastCommand + fmt.Sprintf(" %s=%s", paramNameValue[0], strings.Replace(paramNameValue[1], " ", "", -1))
		}
	}
	return commands, nil
}

func readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}
