package main

import (
	"fmt"
	"strings"
)

// parseSearch parses a search from a query string.
// A query string has a form similar to a search query of the Github REST API;
// it consists of keywords and qualifiers separated by whitespace.
// A qualifier is a string of the form "param:value". A keyword is a plain string.
// An example query: "database crash assignee:oliver"
// Another: "state:closed panic fatal"
func parseSearch(query string) (map[string]string, error) {
	search := make(map[string]string)
	for _, field := range strings.Fields(query) {
		arg := strings.SplitN(field, ":", 2)
		if len(arg) == 1 {
			// concatenate keywords with spaces between each
			search["search"] = search["search"] + " " + arg[0]
			continue
		}
		switch arg[0] {
		case "state", "assignee":
			search[arg[0]] = arg[1]
		default:
			return nil, fmt.Errorf("unknown qualifier %s", arg[0])
		}
	}
	return search, nil
}
