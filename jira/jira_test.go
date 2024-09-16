package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestIssue(t *testing.T) {
	f, err := os.Open("issue.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var issue Issue
	if err := json.NewDecoder(f).Decode(&issue); err != nil {
		t.Fatal(err)
	}
}

func TestPrint(t *testing.T) {
	f, err := os.Open("issue.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var issue Issue
	if err := json.NewDecoder(f).Decode(&issue); err != nil {
		t.Fatal(err)
	}
	printIssue(os.Stdout, &issue)
}
