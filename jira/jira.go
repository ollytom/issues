package jira

import (
	"encoding/json"
	"fmt"
	"time"
)

const timestamp = "2006-01-02T15:04:05.999-0700"

type Issue struct {
	ID       string // TODO(otl): int?
	URL      string
	Key      string
	Reporter User
	Assignee User
	Summary  string
	Status   struct {
		Name string `json:"name"`
	} `json:"status"`
	Description string
	Project     Project
	Created     time.Time
	Updated     time.Time
	Comments    []Comment
	Links       []Issue
	Subtasks    []Issue
}

type Project struct {
	ID string `json:"id"` // TODO(otl): int?
	// Name string `json:"name"`
	Key string `json:"key"`
	URL string `json:"self"`
}

type Comment struct {
	ID           string    `json:"id"` // TODO(otl): int?
	URL          string    `json:"self"`
	Body         string    `json:"body"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
	Author       User      `json:"author"`
	UpdateAuthor User      `json:"updateAuthor"`
}

func (c *Comment) UnmarshalJSON(b []byte) error {
	type alias Comment
	aux := &struct {
		Created string `json:"created"`
		Updated string `json:"updated"`
		*alias
	}{
		alias: (*alias)(c),
	}
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	var err error
	c.Created, err = time.Parse(timestamp, aux.Created)
	if err != nil {
		return fmt.Errorf("parse created time: %w", err)
	}
	c.Updated, err = time.Parse(timestamp, aux.Updated)
	if err != nil {
		return fmt.Errorf("parse updated time: %w", err)
	}
	return nil
}

type User struct {
	Name        string `json:"name"`
	Email       string `json:"emailAddress"`
	DisplayName string `json:"displayName"`
}

func (u User) String() string {
	if u.DisplayName == "" {
		return u.Email
	}
	return fmt.Sprintf("%s <%s>", u.DisplayName, u.Email)
}

func (issue *Issue) UnmarshalJSON(b []byte) error {
	aux := &struct {
		ID     string
		Self   string
		Key    string
		Fields json.RawMessage
	}{}
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	issue.ID = aux.ID
	issue.URL = aux.Self
	issue.Key = aux.Key

	type alias Issue
	iaux := &struct {
		Created    string
		Updated    string
		Comment    map[string]json.RawMessage
		IssueLinks []struct {
			InwardIssue  *Issue
			OutwardIssue *Issue
		}
		*alias
	}{
		alias: (*alias)(issue),
	}
	if err := json.Unmarshal(aux.Fields, iaux); err != nil {
		return err
	}

	var err error
	if iaux.Created != "" {
		issue.Created, err = time.Parse(timestamp, iaux.Created)
		if err != nil {
			return fmt.Errorf("created time: %w", err)
		}
	}
	if iaux.Updated != "" {
		issue.Updated, err = time.Parse(timestamp, iaux.Updated)
		if err != nil {
			return fmt.Errorf("updated time: %w", err)
		}
	}
	if bb, ok := iaux.Comment["comments"]; ok {
		if err := json.Unmarshal(bb, &issue.Comments); err != nil {
			return fmt.Errorf("unmarshal comments: %w", err)
		}
	}
	for _, l := range iaux.IssueLinks {
		if l.InwardIssue != nil {
			issue.Links = append(issue.Links, *l.InwardIssue)
		}
		if l.OutwardIssue != nil {
			issue.Links = append(issue.Links, *l.OutwardIssue)
		}
	}
	return nil
}
