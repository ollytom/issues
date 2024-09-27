package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

const debug = false

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
