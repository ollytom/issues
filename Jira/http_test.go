package main

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"testing"
	"testing/fstest"
)

func handleComment(w http.ResponseWriter, req *http.Request) {
	id := path.Base(req.URL.Path)
	http.ServeFile(w, req, "testdata/comment/"+id)
}

func TestGet(t *testing.T) {
	srv := newFakeServer("testdata")
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{apiRoot: u}

	project := "TEST"
	issue := "TEST-1"
	comment := "69"
	if _, err := client.Project(project); err != nil {
		t.Fatalf("get project %s: %v", project, err)
	}
	if _, err := client.Issues(project); err != nil {
		t.Fatalf("get %s issues: %v", project, err)
	}
	if _, err := client.Issue(issue); err != nil {
		t.Fatalf("get issue %s: %v", issue, err)
	}
	c, err := client.Comment(issue, comment)
	if err != nil {
		t.Fatalf("get comment %s from %s: %v", comment, issue, err)
	}
	if c.ID != comment {
		t.Fatalf("wanted comment id %s, got %s", comment, c.ID)
	}

	fsys := &FS{client: client}
	f, err := fsys.Open("TEST/1/69")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, f); err != nil {
		t.Fatal(err)
	}
	f.Close()

	expected := []string{
		"TEST",
		"TEST/1",
		"TEST/1/issue",
		"TEST/1/69",
	}
	if err := fstest.TestFS(fsys, expected...); err != nil {
		t.Error(err)
	}
}
