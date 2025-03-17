package main

import (
	"fmt"
	"go/doc/comment"
	"strings"
)

// https://jira.atlassian.com/secure/WikiRendererHelpAction.jspa?section=all

func toJTF(content string) string {
	var p comment.Parser
	doc := p.Parse(content)
	buf := &strings.Builder{}
	for _, block := range doc.Content {
		switch v := block.(type) {
		case *comment.Heading:
			fmt.Fprintf(buf, "h3. %s\n", render(v.Text))
		case *comment.Paragraph:
			fmt.Fprintln(buf, render(v.Text))
		case *comment.Code:
			fmt.Fprintf(buf, "{noformat}%s{noformat}\n", v.Text)
		case *comment.List:
			fmt.Fprintln(buf, renderList(v))
		}
		fmt.Fprintln(buf)
	}
	return strings.TrimSpace(buf.String())
}

func renderList(list *comment.List) string {
	buf := &strings.Builder{}
	prefix := "*"
	for _, it := range list.Items {
		if it.Number != "" {
			prefix = "#"
		}
		for _, block := range it.Content {
			// the block is known to be a paragraph
			s := render(block.(*comment.Paragraph).Text)
			fmt.Fprintln(buf, prefix, s)
		}
	}
	return buf.String()
}

func render(text []comment.Text) string {
	buf := &strings.Builder{}
	for _, txt := range text {
		switch v := txt.(type) {
		case comment.Plain:
			s := strings.ReplaceAll(string(v), "\n", " ")
			if strings.HasPrefix(s, "> ") {
				s = strings.ReplaceAll(s, "> ", "")
				fmt.Fprintf(buf, "{quote}%s{quote}", s)
			} else {
				buf.WriteString(s)
			}
		case comment.Italic:
			fmt.Fprintf(buf, "*%s*", v)
		case *comment.Link:
			if v.Auto {
				fmt.Fprintf(buf, "[%s]", v.URL)
			} else {
				title := render(v.Text)
				fmt.Fprintf(buf, "[%s|%s]", title, v.URL)
			}
		case *comment.DocLink:
			// we're not actually printing godoc, so treat
			// any accidental DocLink as plain text.
			buf.WriteString(render(v.Text))
		default:
			fmt.Fprintf(buf, "%v", v)
		}
	}
	return buf.String()
}
