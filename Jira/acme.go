package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
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

var issueKeyExp = regexp.MustCompile("[A-Z]+-[0-9]+")

func (w *awin) Look(text string) bool {
	text = strings.TrimSpace(text)

	var pathname string
	if issueKeyExp.MatchString(text) {
		proj, num, _ := strings.Cut(text, "-")
		pathname = path.Join(proj, num, "issue")
	} else {
		pathname = path.Join(path.Dir(w.name()), text)
		if strings.HasSuffix(text, "/") {
			pathname = path.Join(w.name(), text)
		}
	}
	f, err := w.fsys.Open(pathname)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	} else if err != nil {
		w.Err(err.Error())
		return false
	}

	win, err := acme.New()
	if err != nil {
		w.Err(err.Error())
		return true
	}
	wname := path.Join("/jira", pathname)
	if d, ok := f.(fs.DirEntry); ok {
		if d.IsDir() {
			wname += "/"
		}
	} else {
		stat, err := f.Stat()
		if err != nil {
			w.Err(err.Error())
			return true
		}
		if stat.IsDir() {
			wname += "/"
		}
	}
	win.Name(wname)
	if path.Base(pathname) == "issue" {
		win.Fprintf("tag", "Comment ")
	}
	ww := &awin{win, w.fsys}
	go ww.EventLoop(ww)
	go func() {
		if err := ww.Get(f); err != nil {
			w.Err(err.Error())
		}
		ww.Addr("#0")
		ww.Ctl("dot=addr")
		ww.Ctl("show")
	}()
	return true
}

func (w *awin) Execute(cmd string) bool {
	fields := strings.Fields(strings.TrimSpace(cmd))
	switch fields[0] {
	case "Get":
		if err := w.Get(nil); err != nil {
			w.Err(err.Error())
		}
		return true
	case "Search":
		if len(fields) == 1 {
			return false
		}
		query := strings.Join(fields[1:], " ")
		go newSearch(w.fsys, query)
		return true
	case "Comment":
		if len(fields) > 1 {
			return false
		}
		win, err := acme.New()
		if err != nil {
			w.Err(err.Error())
			return false
		}
		dname := path.Dir(w.name())
		win.Name(path.Join("/jira", dname, "new"))
		win.Fprintf("tag", "Post ")
		a := &awin{win, w.fsys}
		go a.EventLoop(a)
		return true
	case "Post":
		if err := w.postComment(); err != nil {
			w.Errf("post comment: %s", err.Error())
		}
		w.Del(true)
		return true
	}
	return false
}

func (w *awin) Get(f fs.File) error {
	defer w.Ctl("clean")
	fname := path.Clean(w.name())
	if fname == "/" {
		fname = "." // special name for the root file in io/fs
	}
	if f == nil {
		var err error
		f, err = w.fsys.Open(fname)
		if err != nil {
			return err
		}
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	defer f.Close()
	if stat.IsDir() {
		dirs, err := f.(fs.ReadDirFile).ReadDir(-1)
		if err != nil {
			return err
		}
		buf := &strings.Builder{}
		for _, d := range dirs {
			if d.IsDir() {
				fmt.Fprintln(buf, d.Name()+"/")
				continue
			}
			fmt.Fprintln(buf, d.Name())
		}
		w.Clear()
		w.PrintTabbed(buf.String())
		return nil
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read %s: %w", stat.Name(), err)
	}
	w.Clear()
	if _, err := w.Write("body", b); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}

func (w *awin) postComment() error {
	defer w.Ctl("clean")
	body, err := w.ReadAll("body")
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	f, ok := w.fsys.(*FS)
	if !ok {
		return fmt.Errorf("cannot write comment with filesystem type %T", w.fsys)
	}
	elems := strings.Split(w.name(), "/")
	ikey := fmt.Sprintf("%s-%s", elems[0], elems[1])
	return f.client.PostComment(ikey, bytes.NewReader(body))
}

func newSearch(fsys fs.FS, query string) {
	win, err := acme.New()
	if err != nil {
		acme.Errf("new window: %v", err.Error())
		return
	}
	defer win.Ctl("clean")
	win.Name("/jira/search")
	f, ok := fsys.(*FS)
	if !ok {
		win.Errf("cannot search with filesystem type %T", fsys)
	}
	win.PrintTabbed("Search " + query + "\n\n")
	issues, err := f.client.SearchIssues(query)
	if err != nil {
		win.Errf("search %q: %v", query, err)
		return
	}
	win.PrintTabbed(printIssues(issues))
	w := &awin{win, fsys}
	go w.EventLoop(w)
}

const usage string = "usage: Jira [keyfile]"

func readCreds(name string) (username, password string, err error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return "", "", err
	}
	u, p, found := strings.Cut(strings.TrimSpace(string(b)), ":")
	if !found {
		return "", "", fmt.Errorf("missing userpass field separator %q", ":")
	}
	return u, p, nil
}

var hostFlag = flag.String("h", "jira.atlassian.com", "")

func main() {
	flag.Parse()
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("find user config dir: %v", err)
	}
	credPath := path.Join(home, ".config/atlassian/jira")
	if len(flag.Args()) == 1 {
		credPath = flag.Args()[0]
	} else if len(flag.Args()) > 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	user, pass, err := readCreds(credPath)
	if err != nil {
		log.Fatalf("read credentials: %v", err)
	}

	// srv := newFakeServer("testdata")
	// defer srv.Close()

	u, err := url.Parse("https://" + *hostFlag + "/rest/api/2")
	if err != nil {
		log.Fatalf("parse api root url: %v", err)
	}
	fsys := &FS{
		client: &Client{
			debug:    false,
			apiRoot:  u,
			username: user,
			password: pass,
		},
	}

	acme.AutoExit(true)
	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	win.Name("/jira/")
	root := &awin{win, fsys}
	root.Get(nil)
	root.Addr("#0")
	root.Ctl("dot=addr")
	root.Ctl("show")
	win.Ctl("clean")
	root.EventLoop(root)
}
