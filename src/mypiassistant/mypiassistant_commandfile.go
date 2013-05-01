package main

import (
	"path/filepath"
	"utils"
)

func (self *PiAssistant) getCommandFile(fileUrl string) (string, error) {
	ext := filepath.Ext(fileUrl)
	fileName := utils.RandomString(7) + ext
	return utils.DownloadHttpFile(fileUrl, fileName)
}
