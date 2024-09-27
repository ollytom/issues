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

	wname := path.Join("/jira", w.name(), path.Base(fname))
	if stat.IsDir() {
		wname += "/"
	}
	win.Name(wname)
	go func() {
		defer f.Close()
		buf := &bytes.Buffer{}
		if stat.IsDir() {
			dirs, err := f.(fs.ReadDirFile).ReadDir(-1)
			if err != nil {
				win.Err(err.Error())
				return
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
				win.Err(err.Error())
				return
			}
		}
		w := &awin{win, w.fsys}
		if _, err := w.Write("body", buf.Bytes()); err != nil {
			win.Err(err.Error())
			return
		}
		win.Ctl("clean")
		go w.EventLoop(w)
	}()
	return true
}

func (w *awin) Execute(cmd string) bool {
	fields := strings.Fields(strings.TrimSpace(cmd))
	switch fields[0] {
	case "Get":
		return false
	case "Search":
		if len(fields) == 1 {
			return false
		}
		query := strings.Join(fields[1:], " ")
		go newSearch("TODO", query)
		return true
	}
	return false
}

func newSearch(apiRoot, query string) {
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
	_, err = win.Write("body", []byte(printIssues(issues)))
	if err != nil {
		win.Err(err.Error())
	}
}

func main() {
	fsys := &FS{apiRoot: "https://jira.atlassian.com/rest/api/2"}

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
