package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const GitlabHosted string = "https://gitlab.com/api/v4"

type gError struct {
	Message any
}

func (e gError) Error() string {
	switch v := e.Message.(type) {
	case string:
		return v
	}
	return "unknown"
}

type Issue struct {
	ID      int       `json:"iid"`
	Title   string    `json:"title"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
	Closed  time.Time `json:"closed_at"`
	State   string    `json:"state"`
	Author  struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	URL         string   `json:"web_url"`
}

type Client struct {
	*http.Client
	BaseURL string
	Token   string
}

func (c *Client) Issues(project string, search map[string]string) ([]Issue, error) {
	p := path.Join("projects", url.PathEscape(project), "issues")
	resp, err := c.get(p)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return issues, nil
}

func (c *Client) Issue(project string, id int) (*Issue, error) {
	p := path.Join("projects", url.PathEscape(project), "issues", strconv.Itoa(id))
	resp, err := c.get(p)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &issue, nil
}

func (c *Client) Create(project, title, desc string) (*Issue, error) {
	m := map[string]string{
		"title":       title,
		"description": desc,
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	p := path.Join("projects", url.PathEscape(project), "issues")
	resp, err := c.post(p, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode created issue: %w", err)
	}
	return &issue, nil
}

func (c *Client) get(path string) (*http.Response, error) {
	if c.BaseURL == "" {
		c.BaseURL = GitlabHosted
	}
	u := fmt.Sprintf("%s/%s", c.BaseURL, path)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) post(path, contentType string, body io.Reader) (*http.Response, error) {
	if c.BaseURL == "" {
		c.BaseURL = GitlabHosted
	}
	u := fmt.Sprintf("%s/%s", c.BaseURL, path)
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return resp, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://gitlab.com/api/v4"
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	req.Header.Set("Accept", "application/json")
	if *debug {
		log.Println(req.Method, req.URL)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var e gError
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("%s %s: %s: decode error message: %w", req.Method, req.URL, resp.Status, err)
		}
		resp.Body.Close()
		return nil, fmt.Errorf("%s %s: %s: %w", req.Method, req.URL, resp.Status, e)
	}
	return resp, err
}
