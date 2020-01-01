package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

const configFile = "conf.txt"

func configFilePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	directoryPath := filepath.Dir(ex)
	return path.Join(directoryPath, configFile)
}

func readLastSentDate() string {
	dateString, err := ioutil.ReadFile(configFilePath())
	if err != nil {
		fmt.Println("Can't read config:", err)
	}
	return string(dateString)
}

func saveCurrentDate(dateString string) {
	err := ioutil.WriteFile(configFilePath(), []byte(dateString), 0644)
	if err != nil {
		fmt.Println("Can't write config:", err)
	}
}
