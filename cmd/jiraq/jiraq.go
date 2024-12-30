// Command jiraq lists Jira issues matching the provided Jira query.
// Queries must be provided as a single quoted argument in JQL format,
// such as "project = EXAMPLE and status = Done".
//
// Its usage is:
//
//	jiraq [ -u url ] query
//
// The flags are:
//
//	-u url
//		The URL pointing to the root of the JIRA REST API.
//
//
// # Examples
//
// Print an overview of all open tickets in the project "SRE":
//
//	jiraq -u https://company.example.net 'project = SRE and status != done'
//
// Subsequent examples omit the "-u" flag for brevity.
// List all open tickets assigned to yourself in the project "SRE":
//
//	jiraq 'project = SRE and status != done and assignee = currentuser()'
//
// Print issues updated since yesterday:
//
//	query='project = SRE and status != done and updated >= -24h'
// 	jiraexport `jiraq "$query" | awk '{print $1}'`
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"olowe.co/issues/jira"
)

func readJiraAuth() (user, pass string, err error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}
	b, err := os.ReadFile(path.Join(confDir, "atlassian/jira"))
	if err != nil {
		return "", "", err
	}
	b = bytes.TrimSpace(b)
	u, p, ok := strings.Cut(string(b), ":")
	if !ok {
		return "", "", fmt.Errorf(`missing ":" between username and password`)
	}
	return u, p, nil
}

var apiRoot = flag.String("u", "http://[::1]:8080", "base URL for the JIRA API")

const usage = "usage: jiraq [-u url] query"

func init() {
	log.SetPrefix("jiraq: ")
	log.SetFlags(0)
	flag.Parse()
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal(usage)
	}

	user, pass, err := readJiraAuth()
	if err != nil {
		log.Fatalf("read jira auth credentials: %v", err)
	}
	u, err := url.Parse(*apiRoot)
	if err != nil {
		log.Fatalln("parse api url:", err)
	}
	u.Path = path.Join(u.Path, "rest/api/2")
	client := &jira.Client{
		APIRoot:  u,
		Username: user,
		Password: pass,
	}

	issues, err := client.SearchIssues(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}
	for _, is := range issues {
		fmt.Printf("%s-%s\t%s\n", is.Project.Name(), is.Name(), is.Summary)
	}
}
