package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"
	"time"
)

type FS struct {
	apiRoot string
	root    *fid
}

// JRASERVER/1234/issue
// JRASERVER/1234/5678

const (
	ftypeRoot int = iota
	ftypeProject
	ftypeIssue
	ftypeIssueDir
	ftypeComment
)

type fid struct {
	apiRoot string
	name    string
	typ     int
	rd      io.Reader
	parent  *fid

	// May not be set.
	stat *stat

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
	if debug {
		log.Println("stat", f.name)
	}
	if f.stat != nil {
		return f.stat, nil
	}

	switch f.typ {
	case ftypeRoot:
		return &stat{".", int64(len(f.children)), 0o444 | fs.ModeDir, time.Time{}}, nil
	case ftypeProject:
		p, err := getProject(f.apiRoot, f.name)
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		return p, nil
	case ftypeIssueDir, ftypeIssue:
		is, err := getIssue(f.apiRoot, f.issueKey())
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		if f.typ == ftypeIssueDir {
			return is, nil
		}
		return &stat{f.name, int64(len(printIssue(is))), 0o444, is.Updated}, nil
	case ftypeComment:
		c, err := getComment(f.apiRoot, f.issueKey(), f.name)
		if err != nil {
			return nil, &fs.PathError{"stat", f.name, err}
		}
		return c, nil
	}
	err := fmt.Errorf("unexpected fid type %d", f.typ)
	return nil, &fs.PathError{"stat", f.name, err}
}

func (f *fid) Read(p []byte) (n int, err error) {
	if f.rd == nil {
		switch f.typ {
		case ftypeComment:
			c, err := getComment(f.apiRoot, f.issueKey(), f.name)
			if err != nil {
				err = fmt.Errorf("get comment %s: %w", f.issueKey(), err)
				return 0, &fs.PathError{"read", f.name, err}
			}
			f.rd = strings.NewReader(printComment(c))
		case ftypeIssue:
			is, err := getIssue(f.apiRoot, f.issueKey())
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
	return nil
}

func (f *fid) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	if debug {
		log.Println("readdir", f.name)
	}
	if f.children == nil {
		switch f.typ {
		case ftypeRoot:
			return nil, fmt.Errorf("root initialised incorrectly: no dir entries")
		case ftypeProject:
			issues, err := getIssues(f.apiRoot, f.name)
			if err != nil {
				return nil, fmt.Errorf("get issues: %w", err)
			}
			f.children = make([]fs.DirEntry, len(issues))
			for i, issue := range issues {
				f.children[i] = &fid{
					apiRoot: f.apiRoot,
					name:    issue.Name(),
					typ:     ftypeIssueDir,
					parent:  f,
				}
			}
		case ftypeIssueDir:
			issue, err := getIssue(f.apiRoot, f.issueKey())
			if err != nil {
				return nil, fmt.Errorf("get issue %s: %w", f.name, err)
			}
			f.children = make([]fs.DirEntry, len(issue.Comments)+1)
			for i, c := range issue.Comments {
				f.children[i] = &fid{
					apiRoot: f.apiRoot,
					name:    c.ID,
					typ:     ftypeComment,
					rd:      strings.NewReader(printComment(&c)),
					parent:  f,
				}
			}
			f.children[len(f.children)-1] = &fid{
				name:    "issue",
				apiRoot: f.apiRoot,
				typ:     ftypeIssue,
				rd:      strings.NewReader(issue.Summary),
				parent:  f,
				stat:    &stat{"issue", int64(len(printIssue(issue))), 0o444, issue.Updated},
			}
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

	var err error
	if fsys.root == nil {
		fsys.root, err = makeRoot(fsys.apiRoot)
		if err != nil {
			return nil, fmt.Errorf("make root file: %w", err)
		}
	}
	if debug {
		log.Println("open", name)
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

func makeRoot(apiRoot string) (*fid, error) {
	projects, err := getProjects(apiRoot)
	if err != nil {
		return nil, err
	}
	root := &fid{
		apiRoot:  apiRoot,
		name:     ".",
		typ:      ftypeRoot,
		children: make([]fs.DirEntry, len(projects)),
	}
	for i, p := range projects {
		root.children[i] = &fid{
			apiRoot: apiRoot,
			name:    p.Key,
			typ:     ftypeProject,
		}
	}
	return root, nil
}

func find(dir *fid, name string) (*fid, error) {
	if !dir.IsDir() {
		return nil, fs.ErrNotExist
	}
	child := &fid{apiRoot: dir.apiRoot, parent: dir}
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
		ok, err := checkIssue(dir.apiRoot, key)
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
		// we may have already loaded the dir entries (comments)
		// when we loaded the parent (issue).
		/*
			for _, d := range dir.children {
				if d.Name() == name {
					c, ok := d.(*fid)
					if !ok {
						break
					}
					return c, nil
				}
			}
		*/
		ok, err := checkComment(dir.apiRoot, dir.issueKey(), name)
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
