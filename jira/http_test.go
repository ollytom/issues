package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"testing/fstest"
)

func serveFakeList(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var prefix string
		typ := path.Base(dir)
		switch typ {
		case "issue":
			prefix = `{"issues": [`
		case "project":
			prefix = "["
		default:
			http.NotFound(w, req)
			return
		}
		dirs, err := os.ReadDir(dir)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, prefix)
		for i, d := range dirs {
			f, err := os.Open(path.Join(dir, d.Name()))
			if err != nil {
				log.Println(err)
				return
			}
			if _, err := io.Copy(w, f); err != nil {
				log.Printf("copy %s: %v", f.Name(), err)
				f.Close()
			}
			f.Close()
			if i == len(dirs)-1 {
				break
			}
			fmt.Fprintln(w, ",")
		}
		fmt.Fprintln(w, "]}")
	}
}

func handleComment(w http.ResponseWriter, req *http.Request) {
	id := path.Base(req.URL.Path)
	http.ServeFile(w, req, "testdata/comment/"+id)
}

func TestGet(t *testing.T) {
	http.HandleFunc("/project", serveFakeList("testdata/project"))
	http.HandleFunc("/search", serveFakeList("testdata/issue"))
	http.HandleFunc("/issue", serveFakeList("testdata/issue"))
	http.HandleFunc("/issue/TEST-1/comment/", handleComment)
	http.Handle("/", http.FileServer(http.Dir("testdata")))
	srv := httptest.NewServer(nil)
	defer srv.Close()

	project := "TEST"
	issue := "TEST-1"
	comment := "69"
	if _, err := getProject(srv.URL, project); err != nil {
		t.Fatalf("get project %s: %v", project, err)
	}
	if _, err := getIssues(srv.URL, project); err != nil {
		t.Fatalf("get %s issues: %v", project, err)
	}
	if _, err := getIssue(srv.URL, issue); err != nil {
		t.Fatalf("get issue %s: %v", issue, err)
	}
	c, err := getComment(srv.URL, issue, comment)
	if err != nil {
		t.Fatalf("get comment %s from %s: %v", comment, issue, err)
	}
	if c.ID != "69" {
		t.Fatalf("wanted comment id %s, got %s", "69", c.ID)
	}

	fsys := &FS{apiRoot: srv.URL}
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
