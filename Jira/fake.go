package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
)

// newFakeServer returns a fake JIRA server which serves projects,
// issues, and comments from the filesystem tree rooted at root.
// For an example tree, see the testdata directory.
//
// The server provides a limited read-only subset of the JIRA HTTP API
// intended for testing API clients.
// All search requests return a list of every issue, even if the JQL query is invalid.
// Paginated responses are not supported.
func newFakeServer(root string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/project", serveJSONList(path.Join(root, "project")))
	mux.HandleFunc("/search", serveJSONList(path.Join(root, "issue")))
	mux.HandleFunc("/issue", serveJSONList(path.Join(root, "issue")))
	mux.HandleFunc("/issue/", handleIssues(root))
	mux.Handle("/", http.FileServer(http.Dir(root)))
	return httptest.NewServer(mux)
}

func serveJSONList(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		prefix := "["
		if path.Base(dir) == "issue" {
			prefix = `{"issues": [`
		}
		dirs, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, req)
			return
		} else if err != nil {
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

func handleIssues(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if match, _ := path.Match("/issue/*/comment/*", req.URL.Path); match {
			// ignore error; we know pattern is ok.
			file := path.Base(req.URL.Path)
			http.ServeFile(w, req, path.Join(dir, "comment", file))
			return
		}
		http.FileServerFS(os.DirFS(dir)).ServeHTTP(w, req)
	}
}
