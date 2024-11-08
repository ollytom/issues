package main

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

func printIssues(issues []Issue) string {
	buf := &strings.Builder{}
	for _, ii := range issues {
		name := strings.Replace(ii.Key, "-", "/", 1)
		fmt.Fprintf(buf, "%s/issue\t%s\n", name, ii.Summary)
	}
	return buf.String()
}

func printIssue(i *Issue) string {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "From:", i.Reporter)
	fmt.Fprintln(buf, "Date:", i.Created.Format(time.RFC1123Z))
	if i.Assignee.String() != "" {
		fmt.Fprintln(buf, "Assignee:", i.Assignee)
	}
	if u, err := url.Parse(i.URL); err == nil {
		u.Path = path.Join("browse", i.Key)
		fmt.Fprintf(buf, "Archived-At: <%s>\n", u)
	}
	fmt.Fprintf(buf, "Archived-At: <%s>\n", i.URL)
	fmt.Fprintln(buf, "Status:", i.Status.Name)
	if len(i.Links) > 0 {
		s := make([]string, len(i.Links))
		for j := range i.Links {
			s[j] = i.Links[j].Key
		}
		fmt.Fprintln(buf, "References:", strings.Join(s, ", "))
	}
	if len(i.Subtasks) > 0 {
		s := make([]string, len(i.Subtasks))
		for j := range i.Subtasks {
			s[j] = i.Subtasks[j].Key
		}
		fmt.Fprintln(buf, "Subtasks:", strings.Join(s, ", "))
	}
	fmt.Fprintln(buf, "Subject:", i.Summary)
	fmt.Fprintln(buf)

	if i.Description != "" {
		fmt.Fprintln(buf, strings.ReplaceAll(i.Description, "\r", ""))
	}
	if len(i.Comments) == 0 {
		return buf.String()
	}
	fmt.Fprintln(buf)
	for _, c := range i.Comments {
		date := c.Created
		if !c.Updated.IsZero() {
			date = c.Updated
		}
		fmt.Fprintf(buf, "%s\t%s\t%s (%s)\n", c.ID, summarise(c.Body, 36), c.Author.Name, date.Format(time.DateTime))
	}
	return buf.String()
}

func printComment(c *Comment) string {
	buf := &strings.Builder{}
	date := c.Created
	if !c.Updated.IsZero() {
		date = c.Updated
	}
	fmt.Fprintln(buf, "From:", c.Author)
	fmt.Fprintln(buf, "Date:", date.Format(time.RFC1123Z))
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, strings.TrimSpace(c.Body))
	return buf.String()
}

func summarise(body string, length int) string {
	if len(body) < length {
		body = strings.ReplaceAll(body, "\n", " ")
		return strings.TrimSpace(body)
	}
	body = body[:length]
	body = strings.ReplaceAll(body, "\r", "")
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.TrimSpace(body)
	body = strings.ReplaceAll(body, "  ", " ")
	return body + "..."
}
