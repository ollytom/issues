package main

import (
	"fmt"
	"io"
	"strings"
)

func printIssue(w io.Writer, i *Issue) (n int, err error) {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "URL:", i.URL)
	fmt.Fprintln(buf)

	fmt.Fprintln(buf, i.Description)
	fmt.Fprintln(buf)

	for _, c := range i.Comments {
		buf.WriteString("\t")
		printComment(buf, &c)
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
