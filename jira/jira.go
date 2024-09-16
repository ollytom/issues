//
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Issue struct {
	ID          string `json:"id"` // TODO(otl): int?
	URL         string `json:"self"`
	Key         string `json:"key"`
	Description string
	Project     Project
	Comments    []Comment
}

type Project struct {
	ID   string `json:"id"` // TODO(otl): int?
	Name string `json:"name"`
	URL  string `json:"self"`
}

type Comment struct {
	URL          string    `json:"self"`
	ID           string    `json:"id"` // TODO(otl): int?
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
	tstamp := "2006-01-02T15:04:05.999-0700"
	c.Created, err = time.Parse(tstamp, aux.Created)
	if err != nil {
		return fmt.Errorf("parse created time: %w", err)
	}
	c.Updated, err = time.Parse(tstamp, aux.Updated)
	if err != nil {
		return fmt.Errorf("parse updated time: %w", err)
	}
	return nil
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func (issue *Issue) UnmarshalJSON(b []byte) error {
	type alias Issue
	aux := &struct {
		Fields struct {
			Description string
			Comment     []Comment
			Project     Project
		}
		*alias
	}{
		alias: (*alias)(issue),
	}
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	issue.Comments = aux.Fields.Comment
	issue.Description = aux.Fields.Description
	issue.Project = aux.Fields.Project
	return nil
}
