package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

type FS struct {
	client *Client
	root   *fid
}

const (
	ftypeRoot int = iota
	ftypeProject
	ftypeIssue
	ftypeIssueDir
	ftypeComment
)

type fid struct {
	*Client
	name   string
	typ    int
	rd     io.Reader
	parent *fid

	// May be set but only as an optimisation to skip a Stat().
	stat fs.FileInfo

	// directories only
	children []fs.DirEntry
	dirp     int
}

func (f *fid) Name() string { return f.name }
func (f *fid) IsDir() bool  { return f.Type().IsDir() }

func (f *fid) Type() fs.FileMode {
	switch f.typ {
	case ftypeRoot, ftypeProject, ftypeIssueDir:
		return fs.ModeDir
	}
	return 0
}

func (f *fid) Info() (fs.FileInfo, error) { return f.Stat() }

func (f *fid) Stat() (fs.FileInfo, error) {
	if f.Client.debug {
		fmt.Fprintln(os.Stderr, "stat", f.Name())
	}
	if f.stat != nil {
		return f.stat, nil
	}

	switch f.typ {
	case ftypeRoot:
		return &stat{".", int64(len(f.children)), 0o444 | fs.ModeDir, time.Time{}}, nil
	case ftypeProject:
		p, err := f.Project(f.name)
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		return p, nil
	case ftypeIssueDir, ftypeIssue:
		is, err := f.Issue(f.issueKey())
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		if f.typ == ftypeIssueDir {
			f.children = issueChildren(f, is)
			return is, nil
		}
		// optimisation: we might read the file soon so load the contents.
		f.rd = strings.NewReader(printIssue(is))
		return &stat{f.name, int64(len(printIssue(is))), 0o444, is.Updated}, nil
	case ftypeComment:
		c, err := f.Comment(f.issueKey(), f.name)
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		// optimisation: we might read the file soon so load the contents.
		f.rd = strings.NewReader(printComment(c))
		return c, nil
	}
	err := fmt.Errorf("unexpected fid type %d", f.typ)
	return nil, &fs.PathError{"stat", f.name, err}
}

func (f *fid) Read(p []byte) (n int, err error) {
	if f.Client.debug {
		fmt.Fprintln(os.Stderr, "read", f.Name())
	}
	if f.rd == nil {
		switch f.typ {
		case ftypeComment:
			c, err := f.Comment(f.issueKey(), f.name)
			if err != nil {
				err = fmt.Errorf("get comment %s: %w", f.issueKey(), err)
				return 0, &fs.PathError{"read", f.name, err}
			}
			f.rd = strings.NewReader(printComment(c))
		case ftypeIssue:
			is, err := f.Issue(f.issueKey())
			if err != nil {
				err = fmt.Errorf("get issue %s: %w", f.issueKey(), err)
				return 0, &fs.PathError{"read", f.name, err}
			}
			f.rd = strings.NewReader(printIssue(is))
		default:
			var err error
			if f.children == nil {
				f.children, err = f.ReadDir(-1)
				if err != nil {
					return 0, &fs.PathError{"read", f.name, err}
				}
			}
			buf := &strings.Builder{}
			for _, d := range f.children {
				fmt.Fprintln(buf, fs.FormatDirEntry(d))
			}
			f.rd = strings.NewReader(buf.String())
		}
	}
	return f.rd.Read(p)
}

func (f *fid) Close() error {
	f.rd = nil
	f.stat = nil
	return nil
}

func (f *fid) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.Client.debug {
		fmt.Fprintln(os.Stderr, "readdir", f.Name())
	}
	if !f.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	if f.children == nil {
		switch f.typ {
		case ftypeRoot:
			return nil, fmt.Errorf("root initialised incorrectly: no dir entries")
		case ftypeProject:
			issues, err := f.Issues(f.name)
			if err != nil {
				return nil, fmt.Errorf("get issues: %w", err)
			}
			f.children = make([]fs.DirEntry, len(issues))
			for i, issue := range issues {
				f.children[i] = &fid{
					Client: f.Client,
					name:   issue.Name(),
					typ:    ftypeIssueDir,
					parent: f,
				}
			}
		case ftypeIssueDir:
			issue, err := f.Issue(f.issueKey())
			if err != nil {
				return nil, fmt.Errorf("get issue %s: %w", f.name, err)
			}
			f.children = issueChildren(f, issue)
		}
	}

	if f.dirp >= len(f.children) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		f.dirp = len(f.children)
		return f.children, nil
	}

	var err error
	d := f.children[f.dirp:]
	if len(d) >= n {
		d = d[:n]
	} else if len(d) <= n {
		err = io.EOF
	}
	f.dirp += n
	return d, err
}

func issueChildren(parent *fid, is *Issue) []fs.DirEntry {
	kids := make([]fs.DirEntry, len(is.Comments)+1)
	for i, c := range is.Comments {
		kids[i] = &fid{
			Client: parent.Client,
			name:   c.ID,
			typ:    ftypeComment,
			rd:     strings.NewReader(printComment(&c)),
			parent: parent,
			stat:   &c,
		}
	}
	kids[len(kids)-1] = &fid{
		name:   "issue",
		Client: parent.Client,
		typ:    ftypeIssue,
		rd:     strings.NewReader(is.Summary),
		parent: parent,
		stat:   &stat{"issue", int64(len(printIssue(is))), 0o444, is.Updated},
	}
	return kids
}

func (f *fid) issueKey() string {
	// to make the issue key e.g. "EXAMPLE-42"
	// we need the name of the issue (parent name, "42")
	// and the name of the project (the issue's parent's name, "EXAMPLE")
	var project, issueNumber string
	switch f.typ {
	default:
		return ""
	case ftypeComment, ftypeIssue:
		project = f.parent.parent.name
		issueNumber = f.parent.name
	case ftypeIssueDir:
		project = f.parent.name
		issueNumber = f.name
	}
	return project + "-" + issueNumber
}

func (fsys *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{"open", name, fs.ErrInvalid}
	}
	name = path.Clean(name)
	if strings.Contains(name, "\\") {
		return nil, fs.ErrNotExist
	}

	if fsys.root == nil {
		var err error
		fsys.root, err = makeRoot(fsys.client)
		if err != nil {
			return nil, fmt.Errorf("make root file: %w", err)
		}
	}
	if fsys.client.debug {
		fmt.Fprintln(os.Stderr, "open", name)
	}

	if name == "." {
		f := *fsys.root
		return &f, nil
	}

	elems := strings.Split(name, "/")
	if elems[0] == "." && len(elems) > 1 {
		elems = elems[1:]
	}

	f := fsys.root
	for _, elem := range elems {
		dir, err := find(f, elem)
		if err != nil {
			return nil, &fs.PathError{"open", name, err}
		}
		f = dir
	}
	g := *f
	return &g, nil
}

func makeRoot(client *Client) (*fid, error) {
	projects, err := client.Projects()
	if err != nil {
		return nil, err
	}
	root := &fid{
		Client:   client,
		name:     ".",
		typ:      ftypeRoot,
		children: make([]fs.DirEntry, len(projects)),
	}
	for i, p := range projects {
		root.children[i] = &fid{
			Client: client,
			name:   p.Key,
			typ:    ftypeProject,
		}
	}
	return root, nil
}

func find(dir *fid, name string) (*fid, error) {
	if !dir.IsDir() {
		return nil, fs.ErrNotExist
	}
	child := &fid{Client: dir.Client, parent: dir}
	switch dir.typ {
	case ftypeRoot:
		for _, d := range dir.children {
			if d.Name() == name {
				child, ok := d.(*fid)
				if !ok {
					return nil, fmt.Errorf("unexpected dir entry type %T", d)
				}
				return child, nil
			}
		}
		return nil, fs.ErrNotExist
	case ftypeProject:
		key := fmt.Sprintf("%s-%s", dir.name, name)
		ok, err := dir.CheckIssue(key)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fs.ErrNotExist
		}
		child.name = name
		child.typ = ftypeIssueDir
		return child, nil
	case ftypeIssueDir:
		if name == "issue" {
			child.name = name
			child.typ = ftypeIssue
			return child, nil
		}
		ok, err := dir.checkComment(dir.issueKey(), name)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, fs.ErrNotExist
		}
		child.name = name
		child.typ = ftypeComment
		return child, nil
	}
	return nil, fs.ErrNotExist
}
