package main

import (
	"bufio"
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
	"strconv"
	"strings"

	"9fans.net/go/acme"
)

type awin struct {
	*acme.Win
	issue *Issue
	// search query used by the Search command
	query string
}

func (w *awin) name() string {
	buf, err := w.ReadAll("tag")
	if err != nil {
		w.Err(err.Error())
		return ""
	}
	name := strings.Fields(string(buf))[0]
	return path.Base(name)
}

func (w *awin) project() string {
	buf, err := w.ReadAll("tag")
	if err != nil {
		w.Err(err.Error())
		return ""
	}
	name := strings.Fields(string(buf))[0]
	dir := path.Dir(name)
	return strings.TrimPrefix(dir, "/gitlab/")
}

func (w *awin) Execute(cmd string) bool {
	switch cmd {
	case "Get":
		w.load()
		return true
	case "New":
		newIssue(w.project())
		return true
	case "Comment":
		var issueID int
		if w.issue != nil {
			issueID = w.issue.ID
		}
		createComment(issueID, w.project())
		return true
	case "Put":
		buf := &bytes.Buffer{}
		b, err := w.ReadAll("body")
		if err != nil {
			w.Err(err.Error())
			return false
		}
		buf.Write(b)

		switch w.name() {
		case "comment":
			w.Err("put comment not yet implemented")
		case "new":
			w.issue = parseIssue(buf)
			w.issue, err = client.Create(w.project(), w.issue.Title, w.issue.Description)
			w.Name(path.Join("/gitlab", w.project(), strconv.Itoa(w.issue.ID)))
			w.load()
		default:
			err = errors.New("file is not comment or new issue")
		}
		if err != nil {
			w.Err(fmt.Sprintf("put %s: %v", w.name(), err))
			return false
		}
		w.Ctl("clean")
		return true
	}

	if strings.HasPrefix(cmd, "Search") {
		query := strings.TrimSpace(strings.TrimPrefix(cmd, "Search"))
		createSearch(query, w.project())
		return true
	}

	return false
}

func (w *awin) Look(text string) bool {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "#")
	if regexp.MustCompile("^[0-9]+$").MatchString(text) {
		id, err := strconv.Atoi(text)
		if err != nil {
			w.Err(err.Error())
			return false
		}
		name := path.Join("/gitlab", w.project(), strconv.Itoa(id))
		if acme.Show(name) != nil {
			return true
		}
		openIssue(id, w.project())
		return true
	}
	return false
}

func (w *awin) load() {
	if w.name() == "all" || w.query != "" {
		w.loadIssueList()
	} else if regexp.MustCompile("^[0-9]+").MatchString(w.name()) {
		w.loadIssue()
	}
}

func (w *awin) loadIssueList() {
	w.Ctl("dirty")
	defer w.Ctl("clean")

	search := make(map[string]string)
	var err error
	if w.query != "" {
		search, err = parseSearch(w.query)
		if err != nil {
			w.Err(err.Error())
			return
		}
	}
	issues, err := client.Issues(w.project(), search)
	if err != nil {
		w.Err(err.Error())
		return
	}
	w.Clear()
	buf := &bytes.Buffer{}
	printIssueList(buf, issues)
	w.Write("body", buf.Bytes())
	w.Ctl("dot=addr")
}

func (w *awin) loadIssue() {
	w.Ctl("dirty")
	defer w.Ctl("clean")
	id, err := strconv.Atoi(w.name())
	if err != nil {
		w.Err(fmt.Sprintf("parse window name as issue id: %v", err))
		return
	}
	issue, err := client.Issue(w.project(), id)
	if err != nil {
		w.Err(err.Error())
		return
	}
	w.issue = issue
	w.Clear()
	buf := &bytes.Buffer{}
	printIssue(buf, issue)

	/*
		// TODO(otl): we can't load issue notes yet.
		asc := "asc"
		sortAscending := &gitlab.ListIssueNotesOptions{
			Sort: &asc,
		}
		notes, _, err := client.Notes.ListIssueNotes(w.project(), id, sortAscending)
		if err != nil {
			w.Err(err.Error())
		}
		printNotes(buf, notes)
	*/

	w.Write("body", buf.Bytes())
	w.Ctl("dot=addr")
}

func printIssueList(w io.Writer, issues []Issue) {
	for i := range issues {
		fmt.Fprintf(w, "%d\t%s\n", issues[i].ID, issues[i].Title)
	}
}

func printIssue(w io.Writer, issue *Issue) {
	fmt.Fprintln(w, "Title:", issue.Title)
	fmt.Fprintln(w, "State:", issue.State)
	fmt.Fprintln(w, "Author:", issue.Author.Username)
	// TODO(otl): we don't store assignees in Issue yet.
	// fmt.Fprint(w, "Assignee: ")
	// if len(issue.Assignees) > 0 {
	//	var v []string
	//	for _, a := range issue.Assignees {
	//		v = append(v, a.Username)
	//	}
	//	fmt.Fprint(w, strings.Join(v, ", "))
	//}
	//fmt.Fprintln(w)
	fmt.Fprintln(w, "Created:", issue.Created)
	fmt.Fprintln(w, "URL:", issue.URL)
	if !issue.Closed.IsZero() {
		fmt.Fprintln(w, "Closed:", issue.Closed)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, issue.Description)
}

/*
func printNotes(w io.Writer, notes []*gitlab.Note) {
	for i := range notes {
		fmt.Fprintf(w, "%s (%s):\n", notes[i].Author.Username, notes[i].CreatedAt)
		fmt.Fprintln(w)
		fmt.Fprintln(w, notes[i].Body)
		fmt.Fprintln(w)
	}
}
*/

func newIssue(project string) {
	win, err := acme.New()
	if err != nil {
		log.Print(err)
	}
	win.Name(path.Join("/gitlab", project, "new"))
	win.Fprintf("tag", "Put ")
	dummy := &awin{
		Win: win,
	}
	dummy.Write("body", []byte("Title: "))
	go dummy.EventLoop(dummy)
}

func createComment(id int, project string) {
	dummy := &awin{}
	win, err := acme.New()
	if err != nil {
		log.Print(err)
	}
	dummy.Win = win
	dummy.Name("/gitlab/" + project + "/comment")
	win.Fprintf("tag", "Put ")
	body := fmt.Sprintf("To: %d\n\n", id)
	dummy.Write("body", []byte(body))
	go dummy.EventLoop(dummy)
}

func createSearch(query, project string) {
	dummy := &awin{}
	win, err := acme.New()
	if err != nil {
		log.Print(err)
	}
	dummy.Win = win
	dummy.Name("/gitlab/" + project + "/search")
	win.Fprintf("tag", "New Get ")
	dummy.query = query
	dummy.loadIssueList()
	go dummy.EventLoop(dummy)
}

func openIssue(id int, project string) {
	dummy := &awin{}
	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	dummy.Win = win
	win.Name("/gitlab/" + project + "/" + strconv.Itoa(id))
	win.Fprintf("tag", "Comment Get ")
	dummy.loadIssue()
	go dummy.EventLoop(dummy)
}

func openProject(project string) {
	dummy := &awin{}
	win, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	dummy.Win = win
	win.Name("/gitlab/" + project + "/all")
	win.Fprintf("tag", "New Get Search ")
	dummy.loadIssueList()
	dummy.EventLoop(dummy)
	// root window deleted, time to exit
	os.Exit(0)
}

/*
func parseAndPutComment(r io.Reader, project string) error {
	id, body, err := parseIssueNote(r)
	if err != nil {
		return fmt.Errorf("parse issue note: %v", err)
	}
	return putIssueNote(id, project, body)
}
*/

func parseIssueNote(r io.Reader) (id int, body string, err error) {
	sc := bufio.NewScanner(r)
	builder := &strings.Builder{}
	var issue int
	var linenum int
	for sc.Scan() {
		linenum++
		if linenum == 1 {
			text := strings.TrimPrefix(sc.Text(), "To:")
			text = strings.TrimSpace(text)
			id, err := strconv.Atoi(text)
			if err != nil {
				return 0, "", fmt.Errorf("parse issue id: %v", err)
			}
			issue = id
			continue
		}
		if linenum == 2 && sc.Text() == "" {
			// skip the first empty line between header and body
			continue
		}
		builder.WriteString(sc.Text())
		// preserve newline stripped by scanner
		builder.WriteString("\n")
	}
	if sc.Err() != nil {
		return 0, "", err
	}
	return issue, builder.String(), nil
}

/*
func putIssueNote(id int, project, body string) error {
	opt := &gitlab.CreateIssueNoteOptions{
		Body: &body,
	}
	_, _, err := client.Notes.CreateIssueNote(project, id, opt)
	return err
}
*/

func parseIssue(r io.Reader) *Issue {
	var issue Issue
	sc := bufio.NewScanner(r)
	headerDone := false
	buf := &strings.Builder{}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Title:"):
			issue.Title = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
			continue
		}
		if line == "" && !headerDone {
			// hit a blank line, remaining body is the description
			headerDone = true
			continue
		}
		// can't use TrimSpace; we want to keep leading spaces.
		line = strings.TrimRight(line, " \t") // spaces and tabs
		buf.WriteString(line + "\n")          // add back newline stripped by scanner
	}
	if sc.Err() != nil {
		log.Println("parse issue:", sc.Err())
	}
	issue.Description = buf.String()
	return &issue
}

var client *Client
var hFlag = flag.String("h", "", "gitlab hostname")
var tFlag = flag.String("t", "", "personal access token file")
var pFlag = flag.String("p", "gitlab-org/gitlab", "project")
var debug = flag.Bool("d", false, "debug output")

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("Gitlab: ")
	var tokenPath string
	if *tFlag != "" {
		tokenPath = *tFlag
	} else {
		dir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		host := *hFlag
		if host == "" {
			u, err := url.Parse(GitlabHosted)
			if err != nil {
				log.Fatalln("find token: %v", err)
			}
			host = u.Host
		}
		tokenPath = path.Join(dir, "gitlab", host)
	}
	b, err := os.ReadFile(tokenPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalln("read token:", err)
	}
	client = &Client{
		BaseURL: *hFlag,
		Token:   strings.TrimSpace(string(b)),
	}

	project := *pFlag
	go openProject(project)

	select {}
}
