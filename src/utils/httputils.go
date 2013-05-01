package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

func IsHttpUrl(s string) bool {
	httpRegex, _ := regexp.Compile(`^(http|https)://`)
	return httpRegex.Match([]byte(s))
}

func DownloadHttpFile(fileUrl, localFilePath string) (string, error) {
	f, createFileErr := os.Create(localFilePath)
	defer f.Close()
	if createFileErr != nil {
		return "", createFileErr
	}

	resp, downloadFileErr := http.Get(fileUrl)
	defer resp.Body.Close()
	if downloadFileErr != nil {
		return "", downloadFileErr
	}
	_, writeFileErr := io.Copy(f, resp.Body)
	if writeFileErr != nil {
		return "", writeFileErr
	}
	p, _ := filepath.Abs(f.Name())
	return p, nil
}
