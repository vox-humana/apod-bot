package main

import (
	"fmt"
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
	fmt.Println("Full config: ", readConfig())

	if readLastSentDate("test") == "" {
		t.Error("Last date should not be empty")
	}
}
