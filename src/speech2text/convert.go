package speech2text

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"utils"
)

func mp3ToFlac(mp3File string) (string, error) {
	flacPath, _ := filepath.Abs(utils.RandomString(7) + ".flac")
	output, convertErr := exec.Command("ffmpeg", "-i", mp3File, flacPath).Output()
	if convertErr != nil {
		return "", convertErr
	}
	log.Println("convert output: ", string(output))
	return flacPath, nil
}

func downloadFile(fileUrl string) (string, error) {
	ext := filepath.Ext(fileUrl)
	fileName := utils.RandomString(7) + ext
	f, createFileErr := os.Create(fileName)
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
