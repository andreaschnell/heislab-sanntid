package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

type logSettings map[string]string

var LogSettings logSettings
var ch chan string

func Init() {
	LogSettings = loadLogSettings()
	ch = make(chan string)
	go printer()
}

func PrintDbg(s ...interface{}) {
	fileName, _ := getFilename()
	setting := LogSettings[fileName+"Logging"]
	if setting == "DBG" {
		print(fileName, "DBG", s)
	}

}

func PrintErr(s ...interface{}) {
	fileName, _ := getFilename()
	setting := LogSettings[fileName+"Logging"]
	if setting == "INF" || setting == "ERR" {
		print(fileName, "ERR", s)
	}
}

func PrintInf(s ...interface{}) {
	fileName, _ := getFilename()
	setting := LogSettings[fileName+"Logging"]
	if setting == "DBG" || setting == "INF" || setting == "ERR" {
		print(fileName, "INF", s)
	}
}

func print(fileName, setting string, s ...interface{}) {
	var prefix bytes.Buffer

	prefix.WriteString(fileName)

	for i := 1; i < 24-len(fileName); i++ {
		prefix.WriteString(" ")
	}
	prefix.WriteString(setting + "  |")
	t := append([]interface{}{prefix.String()}, s...)

	ch <- fmt.Sprint(t)

}

func loadLogSettings() logSettings {
	settings := make(logSettings)

	raw, err := ioutil.ReadFile("moduleLogSettings.json")
	if err != nil {
		panic(err.Error())
	}
	json.Unmarshal(raw, &settings)

	return settings
}

func printer() {
	for s := range ch {
		fmt.Println(s)
	}
}

func getFilename() (string, error) {
	var ok bool
	_, fileName, _, ok := runtime.Caller(2)
	if !ok {
		return "", fmt.Errorf("N/A")
	}
	nd, nf := filepath.Split(fileName)
	fileName = filepath.Join(filepath.Base(nd), nf)
	deleteString := ""
	for _, v := range fileName {
		deleteString = deleteString + string(v)
		if string(v) == "\\" {
			fileName = strings.ReplaceAll(fileName, deleteString, "")
		}
	}
	fileName = strings.ReplaceAll(fileName, ".go", "")
	return fileName, nil
}
