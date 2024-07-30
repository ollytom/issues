package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestIssues(t *testing.T) {
	f, err := os.Open("issue.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var issue Issue
	if err := json.NewDecoder(f).Decode(&issue); err != nil {
		t.Fatalf("decode issue: %v", err)
	}
	fmt.Println(issue)
}
