package jira

import (
	"io/fs"
	"strings"
	"time"
)

func (is *Issue) Name() string {
	_, number, found := strings.Cut(is.Key, "-")
	if !found {
		return is.Key
	}
	return number
}

func (is *Issue) Size() int64        { return int64(len(printIssue(is))) }
func (is *Issue) Mode() fs.FileMode  { return 0o444 | fs.ModeDir }
func (is *Issue) ModTime() time.Time { return is.Updated }
func (is *Issue) IsDir() bool        { return is.Mode().IsDir() }
func (is *Issue) Sys() any           { return nil }

func (c *Comment) Name() string       { return c.ID }
func (c *Comment) Size() int64        { return int64(len(printComment(c))) }
func (c *Comment) Mode() fs.FileMode  { return 0o444 }
func (c *Comment) ModTime() time.Time { return c.Updated }
func (c *Comment) IsDir() bool        { return c.Mode().IsDir() }
func (c *Comment) Sys() any           { return nil }

func (p *Project) Name() string       { return p.Key }
func (p *Project) Size() int64        { return -1 }
func (p *Project) Mode() fs.FileMode  { return 0o444 | fs.ModeDir }
func (p *Project) ModTime() time.Time { return time.Time{} }
func (p *Project) IsDir() bool        { return p.Mode().IsDir() }
func (p *Project) Sys() any           { return nil }

type stat struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
}

func (s stat) Name() string       { return s.name }
func (s stat) Size() int64        { return s.size }
func (s stat) Mode() fs.FileMode  { return s.mode }
func (s stat) ModTime() time.Time { return s.mtime }
func (s stat) IsDir() bool        { return s.Mode().IsDir() }
func (s stat) Sys() any           { return nil }
