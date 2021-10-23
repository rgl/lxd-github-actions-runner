package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type runner struct {
	owner  string
	repo   string
	token  string
	name   string
	labels []string
	config RunnerConfig
}

func newRunner(owner string, repo string, token string, config RunnerConfig) (*runner, error) {
	exists, err := lxcExists(config.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to check for image existence %s: %w", config.Image, err)
	}
	if !exists {
		return nil, fmt.Errorf("lxc image %s does not exist", config.Image)
	}
	return &runner{
		owner:  owner,
		repo:   repo,
		token:  token,
		name:   fmt.Sprintf("%s-%d", config.Name, 0),
		labels: append(config.Labels, runtime.GOARCH),
		config: config,
	}, nil
}

func (r *runner) Exec() error {
	ctx := context.Background()

	log.Printf("Getting a runner registration token for %s/%s", r.owner, r.repo)
	gh := github.NewClient(
		oauth2.NewClient(ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: r.token})))
	registrationToken, _, err := gh.Actions.CreateRegistrationToken(ctx, r.owner, r.repo)
	if err != nil {
		return fmt.Errorf("failed to create runner registration token: %w", err)
	}

	err = lxcCopy(r.config.Image, r.name)
	if err != nil {
		return fmt.Errorf("failed to start runner %s because copy failed: %w", r.name, err)
	}

	log.Printf("Starting the %s runner", r.name)
	err = lxcStart(r.name)
	if err != nil {
		return fmt.Errorf("failed to start runner %s: %w", r.name, err)
	}

	err = lxcConfigure(r.name, r.owner, r.repo, r.labels, registrationToken.GetToken())
	if err != nil {
		return fmt.Errorf("failed to start runner %s because configure failed: %w", r.name, err)
	}

	log.Printf("Executing the %s runner", r.name)
	err = lxcExecRunner(r.name)
	if err != nil {
		return fmt.Errorf("failed to execute the runner %s: %w", r.name, err)
	}

	return nil
}
