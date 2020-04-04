package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

const configFile = "status.json"

var fullConfigFilePath = ""

type Config struct {
	LastSentDate string
}

func configFilePath() string {
	if len(fullConfigFilePath) > 0 {
		return fullConfigFilePath
	}
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	directoryPath := filepath.Dir(ex)
	return path.Join(directoryPath, configFile)
}

func readConfig() map[string]Config {
	body, err := ioutil.ReadFile(configFilePath())
	if err != nil {
		fmt.Println("Can't read config:", err)
		return map[string]Config{}
	}

	var config map[string]Config
	err = json.Unmarshal(body, &config)
	if err != nil {
		fmt.Println("Can't unmarshal config:", err)
	}
	return config
}

func saveConfig(config map[string]Config) {
	body, err := json.Marshal(config)
	if err != nil {
		fmt.Println("Can't marshal config:", err)
		return
	}
	err = ioutil.WriteFile(configFilePath(), body, 0644)
	if err != nil {
		fmt.Println("Can't write config:", err)
	}
}

func readLastSentDate(service string) string {
	config := readConfig()
	return config[service].LastSentDate
}

func saveCurrentDate(service string, dateString string) {
	config := readConfig()
	var serviceConfig = config[service]
	serviceConfig.LastSentDate = dateString
	config[service] = serviceConfig
	saveConfig(config)
}
