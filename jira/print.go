package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

func printIssue(w io.Writer, i *Issue) (n int, err error) {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "From:", i.Reporter.Name)
	fmt.Fprintln(buf, "URL:", i.URL)
	fmt.Fprintln(buf, "Subject:", i.Summary)
	fmt.Fprintln(buf)

	fmt.Fprintln(buf, i.Description)
	fmt.Fprintln(buf)
	for _, c := range i.Comments {
		date := c.Created
		if !c.Updated.IsZero() {
			date = c.Updated
		}
		fmt.Fprintf(buf, "%s/\t%s\t%s (%s)\n", c.ID, summarise(c.Body), c.Author.Name, date.Format(time.DateTime))
	}
	return w.Write([]byte(buf.String()))
}

func printComment(w io.Writer, c *Comment) (n int, err error) {
	buf := &strings.Builder{}
	date := c.Created
	if !c.Updated.IsZero() {
		date = c.Updated
	}
	fmt.Fprintln(buf, "Date:", date)
	fmt.Fprintln(buf, "From:", c.Author.Name)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, c.Body)
	return w.Write([]byte(buf.String()))
}

func summarise(body string) string {
	max := 36
	if len(body) < max {
		body = strings.ReplaceAll(body, "\n", " ")
		return strings.TrimSpace(body)
	}
	body = body[:max]
	body = strings.ReplaceAll(body, "\r", "")
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.TrimSpace(body)
	body = strings.ReplaceAll(body, "  ", " ")
	return body + "..."
}
