package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const debug = false

type Client struct {
	*http.Client
	debug   bool
	apiRoot *url.URL
}

func (c *Client) Projects() ([]Project, error) {
	u := *c.apiRoot
	u.Path += "/project"

	resp, err := c.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var p []Project

	var p []Project
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return
	return nil, fmt.Errorf("TODO")
}

func (c *Client) getDecode(path string, v any) error {
	u := *c.apiRoot
	resp, err := c.Get(url)
	if err != nil {
		...
	}
	return json.NewDecoder(resp.Body).Decode(&v)
}

func getProjects(apiRoot string) ([]Project, error) {
	u := fmt.Sprintf("%s/project", apiRoot)
	if debug {
		fmt.Fprintln(os.Stderr, "GET", u)
	}
	resp, err := http.Get(u)
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

func getProject(apiRoot, name string) (*Project, error) {
	u := fmt.Sprintf("%s/project/%s", apiRoot, name)
	if debug {
		fmt.Fprintln(os.Stderr, "GET", u)
	}
	resp, err := http.Get(u)
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

func getIssues(apiRoot, project string) ([]Issue, error) {
	q := fmt.Sprintf("project = %q", project)
	return searchIssues(apiRoot, q)
}

func searchIssues(apiRoot, query string) ([]Issue, error) {
	u, err := url.Parse(apiRoot + "/search")
	if err != nil {
		return nil, err
	}
	q := make(url.Values)
	q.Add("jql", query)
	u.RawQuery = q.Encode()
	if debug {
		fmt.Fprintln(os.Stderr, "GET", u)
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
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

func checkIssue(apiRoot, name string) (bool, error) {
	u := fmt.Sprintf("%s/issue/%s", apiRoot, name)
	if debug {
		log.Println("HEAD", u)
	}
	resp, err := http.Head(u)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func getIssue(apiRoot, name string) (*Issue, error) {
	u := fmt.Sprintf("%s/issue/%s", apiRoot, name)
	if debug {
		log.Println("GET", u)
	}
	resp, err := http.Get(u)
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

func checkComment(apiRoot, ikey, id string) (bool, error) {
	u := fmt.Sprintf("%s/issue/%s/comment/%s", apiRoot, ikey, id)
	if debug {
		log.Println("HEAD", u)
	}
	resp, err := http.Head(u)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func getComment(apiRoot string, ikey, id string) (*Comment, error) {
	u := fmt.Sprintf("%s/issue/%s/comment/%s", apiRoot, ikey, id)
	if debug {
		fmt.Fprintln(os.Stderr, "GET", u)
	}
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-ok status: %s", resp.Status)
	}
	defer resp.Body.Close()
	var c Comment
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, fmt.Errorf("decode comment: %w", err)
	}
	return &c, nil
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
