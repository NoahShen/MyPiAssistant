package speech2text

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var Debug = false

const (
	SPEECH_URL = "http://www.google.com/speech-api/v1/recognize?xjerr=1&client=chromium&lang=zh-CN"
)

type speechRespJson struct {
	Id         string       `json:"id,omitempty"`
	Status     int          `json:"status,omitempty"`
	Hypotheses []hypotheses `json:"hypotheses,omitempty"`
}

type hypotheses struct {
	Utterance  string  `json:"utterance,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

func Speech2Text(voiceUrl string) (string, float64, error) {
	if Debug {
		fmt.Printf("Downloading file: %s\n", voiceUrl)
	}
	voiceFilePath, downloadFileErr := downloadFile(voiceUrl)
	defer os.Remove(voiceFilePath)
	if downloadFileErr != nil {
		return "", 0, downloadFileErr
	}
	var flacFile string
	voiceFmt := filepath.Ext(voiceUrl)
	switch strings.ToLower(voiceFmt) {
	case ".mp3":
		if Debug {
			fmt.Printf("Voice format is mp3, starting convert to flac\n")
		}
		p, convertErr := mp3ToFlac(voiceFilePath)
		if convertErr != nil {
			return "", 0, convertErr
		}
		flacFile = p
	case ".flac":
		flacFile = voiceFilePath
	}
	defer os.Remove(flacFile)
	return convertToText(flacFile)
}

func convertToText(voiceFile string) (string, float64, error) {
	bodyBuf := new(bytes.Buffer)
	bodyWriter := multipart.NewWriter(bodyBuf)
	defer bodyWriter.Close()
	fileWriter, _ := bodyWriter.CreateFormFile("uploadfile", voiceFile)

	fh, openErr := os.Open(voiceFile)
	if openErr != nil {
		return "", 0, openErr
	}
	io.Copy(fileWriter, fh)
	speechReq, _ := http.NewRequest("POST", SPEECH_URL, bodyBuf)
	speechReq.Header.Set("Content-Type", "audio/x-flac; rate=16000")
	speechResp, postErr := http.DefaultClient.Do(speechReq)
	if postErr != nil {
		return "", 0, postErr
	}
	defer speechResp.Body.Close()
	bytes, readErr := ioutil.ReadAll(speechResp.Body)
	if readErr != nil {
		return "", 0, readErr
	}
	if Debug {
		fmt.Printf("Speech response json: %s\n", strings.TrimSpace(string(bytes)))
	}
	speechResult := &speechRespJson{}
	if unmarshalErr := json.Unmarshal(bytes, speechResult); unmarshalErr != nil {
		return "", 0, unmarshalErr
	}
	if speechResult.Status != 0 {
		return "", 0, errors.New("speech2text error!")
	}
	h := speechResult.Hypotheses[0]
	return h.Utterance, h.Confidence, nil
}
