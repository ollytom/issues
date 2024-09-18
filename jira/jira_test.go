package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIssue(t *testing.T) {
	names, err := filepath.Glob("issue*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		f, err := os.Open(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		defer f.Close()
		var issue Issue
		if err := json.NewDecoder(f).Decode(&issue); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
