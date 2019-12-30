package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type configuration struct {
	LastSentDate string
}

const configFile = "conf.txt"

func getConfigFilePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	directoryPath := filepath.Dir(ex)
	return path.Join(directoryPath, configFile)
}

func readLastSentDate() string {
	dateString, err := ioutil.ReadFile(getConfigFilePath())
	if err != nil {
		fmt.Println("Can't read config:", err)
	}
	return string(dateString)
}

func saveCurrentDate(dateString string) {
	err := ioutil.WriteFile(getConfigFilePath(), []byte(dateString), 0644)
	if err != nil {
		fmt.Println("Can't write config:", err)
	}
}
