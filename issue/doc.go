// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Issue is a client for reading and updating issues in a GitHub project issue tracker.

	usage: issue [-a] [-e] [-p owner/repo] <query>

Issue runs the query against the given project's issue tracker and
prints a table of matching issues, sorted by issue summary.
The default owner/repo is golang/go.

If multiple arguments are given as the query, issue joins them by
spaces to form a single issue search. These two commands are equivalent:

	issue assignee:rsc author:robpike
	issue "assignee:rsc author:robpike"

Searches are always limited to open issues.

If the query is a single number, issue prints that issue in detail,
including all comments.

# Authentication

Issue expects to find a GitHub "personal access token" in
$HOME/.github-issue-token and will use that token to authenticate
to GitHub when reading or writing issue data.
A token can be created by visiting https://github.com/settings/tokens/new.
The token only needs the 'repo' scope checkbox, and optionally 'private_repo'
if you want to work with issue trackers for private repositories.
It does not need any other permissions.
The -token flag specifies an alternate file from which to read the token.

# Acme Editor Integration

If the -a flag is specified, issue runs as a collection of acme windows
instead of a command-line tool. In this mode, the query is optional.
If no query is given, issue uses "state:open".

This mode requires an existing plumb port called "githubissue", which can
be created with the following plumbing rule:

	plumb to	githubissue

There are three kinds of acme windows: issue, issue creation, issue list,
search result, and milestone list.

The following text forms can be looked for (right clicked on)
and open a window (or navigate to an existing one).

	nnnn			issue #nnnn
	#nnnn			issue #nnnn
	all			the issue list
	milestone(s)		the milestone list
	<milestone-name>	the named milestone (e.g., Go1.5)

Executing "New" opens an issue creation window.

Executing "Search <query>" opens a new window showing the
results of that search.

# Issue Window

An issue window, opened by loading an issue number,
displays full detail about an issue, a header followed by each comment.
For example:

	Title: time: Duration should implement fmt.Formatter
	State: closed
	Assignee: robpike
	Closed: 2015-01-08 05:20:00
	Labels: release-none repo-main size-m
	Milestone:
	URL: https://github.com/golang/go/issues/8786

	Reported by dsymonds (2014-09-21 23:02:50)

		It'd be nice if http://play.golang.org/p/KCnUQOPyol
		printed "[+3us]", which would require time.Duration
		implementing fmt.Formatter to get the '+' flag.

	Comment by rsc (2015-01-08 05:17:06)

		time must not depend on fmt.

Executing "Get" reloads the issue data.

Executing "Put" updates an issue. It saves any changes to the issue header
and, if any text has been entered between the header and the "Reported by" line,
posts that text as a new comment. If both succeed, Put then reloads the issue data.
The "Closed" and "URL" headers cannot be changed.

# Issue Creation Window

An issue creation window, opened by executing "New", is like an issue window
but displays only an empty issue template:

	Title:
	Assignee:
	Labels:
	Milestone:

	<describe issue here>

Once the template has been completed (only the title is required), executing "Put"
creates the issue and converts the window into a issue window for the new issue.

# Issue List Window

An issue list window displays a list of all open issue numbers and titles.
If the project has any open milestones, they are listed in a header line.
For example:

	Milestones: Go1.4.1 Go1.5 Go1.5Maybe

	9027	archive/tar: round-trip of Header misses values
	8669	archive/zip: not possible to a start writing zip at offset other than zero
	8359	archive/zip: not possible to specify deflate compression level
	...

As in any window, right clicking on an issue number opens a window for that issue.

# Search Result Window

A search result window, opened by executing "Search <query>", displays a list of issues
matching a search query. It shows the query in a header line. For example:

	Search author:rsc

	9131	bench: no documentation
	599	cmd/5c, 5g, 8c, 8g: make 64-bit fields 64-bit aligned
	6699	cmd/5l: use m to store div/mod denominator
	4997	cmd/6a, cmd/8a: MOVL $x-8(SP) and LEAL x-8(SP) are different
	...

Executing "Sort" in a search result window toggles between sorting by title
and sorting by decreasing issue number.

# Bulk Edit Window

Executing "Bulk" in an issue list or search result window opens a new
bulk edit window applying to the displayed issues. If there is a non-empty
text selection in the issue list or search result list, the bulk edit window
is restricted to issues in the selection.

The bulk edit window consists of a metadata header followed by a list of issues, like:

	State: open
	Assignee:
	Labels:
	Milestone: Go1.4.3

	10219	cmd/gc: internal compiler error: agen: unknown op
	9711	net/http: Testing timeout on Go1.4.1
	9576	runtime: crash in checkdead
	9954	runtime: invalid heap pointer found in bss on openbsd/386

The metadata header shows only metadata shared by all the issues.
In the above example, all four issues are open and have milestone Go1.4.3,
but they have no common labels nor a common assignee.

The bulk edit applies to the issues listed in the window text; adding or removing
issue lines changes the set of issues affected by Get or Put operations.

Executing "Get" refreshes the metadata header and issue summaries.

Executing "Put" updates all the listed issues. It applies any changes made to
the metadata header and, if any text has been entered between the header
and the first issue line, posts that text as a comment. If all operations succeed,
Put then refreshes the window as Get does.

# Milestone List Window

The milestone list window, opened by loading any of the names
"milestone", "Milestone", or "Milestones", displays the open project
milestones, sorted by due date, along with the number of open issues in each.
For example:

	2015-01-15	Go1.4.1		1
	2015-07-31	Go1.5		215
	2015-07-31	Go1.5Maybe	5

Loading one of the listed milestone names opens a search for issues
in that milestone.

# Alternate Editor Integration

The -e flag enables basic editing of issues with editors other than acme.
The editor invoked is $VISUAL if set, $EDITOR if set, or else ed.
Issue prepares a textual representation of issue data in a temporary file,
opens that file in the editor, waits for the editor to exit, and then applies any
changes from the file to the actual issues.

When <query> is a single number, issue -e edits a single issue.
See the “Issue Window” section above.

If the <query> is the text "new", issue -e creates a new issue.
See the “Issue Creation Window” section above.

Otherwise, for general queries, issue -e edits multiple issues in bulk.
See the “Bulk Edit Window” section above.

# JSON Output

The -json flag causes issue to print the results in JSON format
using these data structures:

	type Issue struct {
		Number    int
		Ref       string
		Title     string
		State     string
		Assignee  string
		Closed    time.Time
		Labels    []string
		Milestone string
		URL       string
		Reporter  string
		Created   time.Time
		Text      string
		Comments  []*Comment
		Reactions Reactions
	}

	type Comment struct {
		Author    string
		Time      time.Time
		Text      string
		Reactions Reactions
	}

	type Reactions struct {
		PlusOne   int
		MinusOne  int
		Laugh     int
		Confused  int
		Heart     int
		Hooray    int
		Rocket    int
		Eyes      int
	}

If asked for a specific issue, the output is an Issue with Comments.
Otherwise, the result is an array of Issues without Comments.
*/
package main
