package speech2text

import (
	"testing"
)

func TestSpeech2Text(t *testing.T) {
	text, c, e := Speech2Text("https://imo.im/fd/E/864hokp84r/voiceim.mp3")
	if e != nil {
		t.Fatal(e)
	}
	t.Log(text)
	t.Log(c)
}

func NoTestDownloadFile(t *testing.T) {
	f, err := downloadFile("https://imo.im/fd/E/864hokp84r/voiceim.mp3")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(f)
}
