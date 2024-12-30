// Command jiraexport prints the named Jira issues, and their comments,
// RFC 5322 mail message format (email).
//
// Usage:
// 	jiraexport [ -d duration ] issue [ ... ]
//
// The options are:
//
//	-d duration
//		Exclude any issues and comments unmodified since duration.
//		Duration may be given in the format accepted by time.ParseDuration.
//		For example, 24h (24 hours). The default is 7 days.
//
// # Example
//
// Print the last day's updates to tickets SRE-1234 and SRE-5678:
//
//	jiraexport -d 24h SRE-1234 SRE-5678
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/mail"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

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

const usage string = "jiraexport [-d duration] [-u url] issue [...]"

var since = flag.Duration("d", 7*24*time.Hour, "exclude activity older than this duration")
var apiRoot = flag.String("u", "http://[::1]:8080", "base URL for the JIRA API")

func init() {
	log.SetFlags(0)
	log.SetPrefix("")
	flag.Parse()
}

func main() {
	if len(flag.Args()) == 0 {
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
	jclient := &jira.Client{
		APIRoot:  u,
		Username: user,
		Password: pass,
		Debug:    false,
	}
	fsys := &jira.FS{Client: jclient}

	for _, arg := range flag.Args() {
		proj, num, ok := strings.Cut(arg, "-")
		if !ok {
			log.Println("bad issue name: missing - separator")
			continue
		}
		dir := path.Join(proj, num)
		f, err := fsys.Open(path.Join(dir, "issue"))
		if err != nil {
			log.Println(err)
			continue
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			log.Println(err)
			continue
		}
		if time.Since(info.ModTime()) >= *since {
			f.Close()
			continue
		}
		msg, err := mail.ReadMessage(f)
		if err != nil {
			f.Close()
			log.Println(err)
			continue
		}
		// fmt.Println("From nobody", info.ModTime().Format(time.ANSIC))
		fmt.Println("Subject:", msg.Header.Get("Subject"))
		/*
			if _, err := io.Copy(os.Stdout, f); err != nil {
				f.Close()
				log.Println(err)
				continue
			}
		*/
		fmt.Println()
		f.Close()

		dents, err := fs.ReadDir(fsys, dir)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, d := range dents {
			if d.Name() == "issue" {
				continue // already done
			}
			info, err := d.Info()
			if err != nil {
				log.Println(err)
				continue
			}
			if time.Since(info.ModTime()) >= *since {
				continue
			}
			f, err := fsys.Open(path.Join(dir, d.Name()))
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Println("From nobody", info.ModTime().Format(time.ANSIC))
			if _, err := io.Copy(os.Stdout, f); err != nil {
				log.Println(err)
			}
			f.Close()
			fmt.Println()
		}
	}
}
