package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

type Client struct {
	*http.Client
	debug              bool
	username, password string
	apiRoot            *url.URL
}

func (c *Client) Projects() ([]Project, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "project")
	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status: %s", resp.Status)
	}
	defer resp.Body.Close()
	var p []Project
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	return p, nil
}

func (c *Client) Project(name string) (*Project, error) {
	u := fmt.Sprintf("%s/project/%s", c.apiRoot, name)
	resp, err := c.get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status: %s", resp.Status)
	}
	defer resp.Body.Close()
	var p Project
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	return &p, nil
}

func (c *Client) Issues(project string) ([]Issue, error) {
	q := fmt.Sprintf("project = %q", project)
	return c.SearchIssues(q)
}

func (c *Client) SearchIssues(query string) ([]Issue, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "search")
	q := make(url.Values)
	q.Add("jql", query)
	u.RawQuery = q.Encode()
	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("bad query")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status: %s", resp.Status)
	}
	t := struct {
		Issues []Issue
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	return t.Issues, nil
}

func (c *Client) CheckIssue(name string) (bool, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "issue", name)
	req, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err != nil {
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func (c *Client) Issue(name string) (*Issue, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "issue", name)
	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status: %s", resp.Status)
	}
	defer resp.Body.Close()
	var is Issue
	if err := json.NewDecoder(resp.Body).Decode(&is); err != nil {
		return nil, fmt.Errorf("decode issue: %w", err)
	}
	return &is, nil
}

func (c *Client) checkComment(ikey, id string) (bool, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "issue", ikey, "comment", id)
	req, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err != nil {
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func (c *Client) Comment(ikey, id string) (*Comment, error) {
	u := *c.apiRoot
	u.Path = path.Join(u.Path, "issue", ikey, "comment", id)
	resp, err := c.get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var com Comment
	if err := json.NewDecoder(resp.Body).Decode(&com); err != nil {
		return nil, fmt.Errorf("decode comment: %w", err)
	}
	return &com, nil
}

func (c *Client) PostComment(issueKey string, body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	cm := Comment{Body: string(b)}
	hbody, err := json.Marshal(&cm)
	if err != nil {
		return fmt.Errorf("to json: %w", err)
	}
	u := fmt.Sprintf("%s/issue/%s/comment", c.apiRoot, issueKey)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(hbody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("non-ok status: %s", resp.Status)
	}
	return nil

}

func Create(apiRoot string, issue Issue) (*Issue, error) {
	b, err := json.Marshal(&issue)
	if err != nil {
		return nil, fmt.Errorf("to json: %w", err)
	}
	u := apiRoot + "/issue"
	resp, err := http.Post(u, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status %s", resp.Status)
	}
	var i Issue
	if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
		return nil, fmt.Errorf("decode created issue: %w", err)
	}
	return &i, nil
}

func CreateComment(apiRoot, issueKey string, body io.Reader) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	c := Comment{Body: string(b)}
	hbody, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("to json: %w", err)
	}
	u := fmt.Sprintf("%s/issue/%s/comment", apiRoot, issueKey)
	resp, err := http.Post(u, "application/json", bytes.NewReader(hbody))
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("non-ok status: %s", resp.Status)
	}
	return nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.debug {
		fmt.Fprintln(os.Stderr, req.Method, req.URL)
	}
	return c.Do(req)
}
