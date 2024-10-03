package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"9fans.net/go/acme"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("Jira: ")
}

type awin struct {
	*acme.Win
	fsys fs.FS
}

func (w *awin) name() string {
	b, err := w.ReadAll("tag")
	if err != nil {
		w.Err(err.Error())
		return ""
	}
	fields := strings.Fields(string(b))
	return strings.TrimPrefix(fields[0], "/jira/")
}

func (w *awin) Look(text string) bool {
	text = strings.TrimSpace(text)
	fname := path.Join(path.Dir(w.name()), text)
	if strings.HasSuffix(text, "/") {
		fname = path.Join(w.name(), text)
	}
	f, err := w.fsys.Open(fname)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	} else if err != nil {
		w.Err(err.Error())
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		w.Err(err.Error())
		return true
	}

	win, err := acme.New()
	if err != nil {
		w.Err(err.Error())
		return true
	}

	wname := path.Join("/jira", fname)
	if stat.IsDir() {
		wname += "/"
	}
	win.Name(wname)
	ww := &awin{win, w.fsys}
	go ww.EventLoop(ww)
	go func() {
		if err := ww.Get(); err != nil {
			w.Err(err.Error())
		}
	}()
	return true
}

func (w *awin) Execute(cmd string) bool {
	fields := strings.Fields(strings.TrimSpace(cmd))
	switch fields[0] {
	case "Get":
		if err := w.Get(); err != nil {
			w.Err(err.Error())
			return false
		}
	case "Search":
		if len(fields) == 1 {
			return false
		}
		query := strings.Join(fields[1:], " ")
		go newSearch(w.fsys, APIRoot, query)
		return true
	}
	return false
}

func (w *awin) Get() error {
	w.Ctl("dirty")
	defer w.Ctl("clean")
	f, err := w.fsys.Open(path.Clean(w.name()))
	if err != nil {
		return err
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	defer f.Close()
	buf := &bytes.Buffer{}
	if stat.IsDir() {
		dirs, err := f.(fs.ReadDirFile).ReadDir(-1)
		if err != nil {
			return err
		}
		for _, d := range dirs {
			if d.IsDir() {
				fmt.Fprintln(buf, d.Name()+"/")
				continue
			}
			fmt.Fprintln(buf, d.Name())
		}
	} else {
		if _, err := io.Copy(buf, f); err != nil {
			return fmt.Errorf("copy %s: %w", stat.Name(), err)
		}
	}
	w.Clear()
	if _, err := w.Write("body", buf.Bytes()); err != nil {
		return fmt.Errorf("write %s: %w", "body", err)
	}
	return nil
}

func newSearch(fsys fs.FS, apiRoot, query string) {
	win, err := acme.New()
	if err != nil {
		acme.Errf("new window: %v", err.Error())
		return
	}
	win.Name("/jira/search")
	issues, err := searchIssues(apiRoot, query)
	if err != nil {
		win.Errf("search %q: %v", query, err)
		return
	}
	if err := win.Fprintf("body", "Search %s\n\n", query); err != nil {
		win.Err(err.Error())
		return
	}
	_, err = win.Write("body", []byte(printIssues(issues)))
	if err != nil {
		win.Err(err.Error())
	}
	w := &awin{win, fsys}
	go w.EventLoop(w)
}

var APIRoot = "https://jira.atlassian.com/api/rest/2"

func main() {
	srv := newFakeServer("testdata")
	defer srv.Close()
	APIRoot = srv.URL
	fsys := &FS{apiRoot: srv.URL}

	acme.AutoExit(true)
	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	root := &awin{win, fsys}

	dirs, err := fs.ReadDir(fsys, ".")
	if err != nil {
		log.Fatal(err)
	}
	buf := &bytes.Buffer{}
	for _, d := range dirs {
		fmt.Fprintln(buf, d.Name()+"/")
	}
	if _, err := root.Write("body", buf.Bytes()); err != nil {
		log.Fatal(err)
	}
	root.Name("/jira/")
	win.Ctl("clean")
	root.EventLoop(root)
	os.Exit(0)
}
