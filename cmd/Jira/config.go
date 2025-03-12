package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	BaseURL  *url.URL
	Username string
	Password string
}

func readConfig(name string) (*Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var conf Config
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		} else if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("key %s: expected whitespace after configuration key", k)
		} else if v == "" {
			return nil, fmt.Errorf("key %s: missing parameter", k)
		}
		switch k {
		case "url":
			u, err := url.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("parse base url: %w", err)
			}
			conf.BaseURL = u
		case "username":
			conf.Username = v
		case "password":
			conf.Password = v
		default:
			return nil, fmt.Errorf("unknown configuration key %q", k)
		}
	}
	return &conf, sc.Err()
}
