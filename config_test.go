package main

import (
	"os"
	"testing"
)

func TestEmptyConfig(t *testing.T) {
	fullConfigFilePath = "test-config.json"
	defer os.Remove(fullConfigFilePath)

	if readLastSentDate("test") != "" {
		t.Error("Last date should be empty")
	}

	saveCurrentDate("test", "2020-04-04")
	config := readConfig()
	if len(config) == 0 {
		t.Error("Saved config shouldn't be empty")
	}

	if readLastSentDate("test") == "" {
		t.Error("Last date should not be empty")
	}
}
