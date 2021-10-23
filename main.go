// Copyright 2021 Rui Lopes (ruilopes.com). All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Repository string
	Runner     RunnerConfig
}

type RunnerConfig struct {
	Name   string
	Image  string
	Labels []string
}

func LoadConfig(path string) (*Config, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from %s: %w", path, err)
	}
	var config Config
	err = yaml.Unmarshal(contents, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from %s: %w", path, err)
	}
	return &config, nil
}

func main() {
	configPath := flag.String("config", "/etc/lxd-ghar/config.yml", "Configuration file path")

	flag.Parse()

	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config %s: %v", *configPath, err)
	}
	repository, err := url.Parse(config.Repository)
	if err != nil {
		log.Fatalf("failed to parse config repository %s: %s", config.Repository, err)
	}
	if repository.Scheme != "https" {
		log.Fatalf("failed to parse config repository %s: scheme %s is not https", config.Repository, repository.Scheme)
	}
	repositorySegments := strings.Split(repository.Path, "/")
	if len(repositorySegments) < 2 {
		log.Fatalf("failed to parse config repository %s: not enough segments to extract the owner and repo name", config.Repository)
	}
	owner := repositorySegments[1]
	repo := strings.Join(repositorySegments[2:], "/")

	token := os.Getenv("GITHUB_TOKEN")
	if err != nil {
		log.Fatalf("failed to get the token from the GITHUB_TOKEN environment variable: %s", err)
	}

	runner, err := newRunner(owner, repo, token, config.Runner)
	if err != nil {
		log.Fatalf("failed to create runner: %s", err)
	}

	err = runner.Exec()
	if err != nil {
		log.Fatalf("failed to exec runner: %s", err)
	}
}
