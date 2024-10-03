package main

import (
	"fmt"
	"strings"
	"time"
)

func printIssues(issues []Issue) string {
	buf := &strings.Builder{}
	for _, ii := range issues {
		name := strings.Replace(ii.Key, "-", "/", 1)
		fmt.Fprintf(buf, "%s/\t%s\n", name, ii.Summary)
	}
	return buf.String()
}

func printIssue(i *Issue) string {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "From:", i.Reporter.Name)
	fmt.Fprintln(buf, "URL:", i.URL)
	fmt.Fprintln(buf, "Date:", i.Updated.Format(time.RFC1123Z))
	fmt.Fprintln(buf, "Status:", i.Status.Name)
	fmt.Fprintln(buf, "Subject:", i.Summary)
	fmt.Fprintln(buf)

	fmt.Fprintln(buf, i.Description)
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
	fmt.Fprintln(buf, "From:", c.Author.Name)
	fmt.Fprintln(buf, "Date:", date.Format(time.RFC1123Z))
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, c.Body)
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
