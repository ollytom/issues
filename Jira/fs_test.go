package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"testing"
)

func TestIssueName(t *testing.T) {
	f, err := os.Open("testdata/issue/TEST-1")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var issue Issue
	if err := json.NewDecoder(f).Decode(&issue); err != nil {
		t.Fatal(err)
	}
	want := "1"
	if issue.Name() != want {
		t.Errorf("issue.Name() = %q, want %q", issue.Name(), want)
	}
}

func TestIssueKey(t *testing.T) {
	comment := &fid{name: "69", typ: ftypeComment}
	issue := &fid{name: "issue", typ: ftypeIssue}
	issueDir := &fid{name: "1", typ: ftypeIssueDir, children: []fs.DirEntry{issue, comment}}
	project := &fid{name: "TEST", typ: ftypeProject, children: []fs.DirEntry{issueDir}}

	comment.parent = issueDir
	issue.parent = issueDir
	issueDir.parent = project

	want := "TEST-1"
	for _, f := range []*fid{comment, issue, issueDir} {
		if f.issueKey() != want {
			t.Errorf("fid %s issueKey = %q, want %q", f.name, f.issueKey(), want)
		}
	}
}
