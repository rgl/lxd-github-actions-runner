package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"

	"github.com/alessio/shellescape"
)

type lxcExecError struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func (err *lxcExecError) Error() string {
	return fmt.Sprintf("failed to exec lxc: exitCode=%d stdout=%s stderr=%s", err.ExitCode, err.Stdout, err.Stderr)
}

func lxc(args ...string) (string, error) {
	return lxcWithInput("", args...)
}

func lxcWithInput(input string, args ...string) (string, error) {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("lxc", args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()

	if err != nil {
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ProcessState.ExitCode()
		}
		return "", &lxcExecError{
			ExitCode: exitCode,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

func lxcExists(name string) (bool, error) {
	output, err := lxc("list", name, "--format", "csv")
	if err != nil {
		return false, fmt.Errorf("failed to find image %s: %w", name, err)
	}
	return output != "", nil
}

func lxcExecRunner(name string) error {
	path, err := exec.LookPath("lxc")
	if err != nil {
		return fmt.Errorf("failed to find lxc: %w", err)
	}
	return syscall.Exec(path, []string{"lxc", "exec", name, "--", "su", "-l", "-s", "/home/ghar/runner/run.sh", "ghar"}, []string{})
}

func lxcCopy(from, to string) error {
	// delete the "to" container if it exists.
	exists, err := lxcExists(to)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s because list failed: %w", from, to, err)
	}
	if exists {
		log.Printf("Deleting the existing %s container", to)
		_, err := lxc("delete", to, "--force")
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s because delete failed: %w", from, to, err)
		}
	}

	// clone the existing container into a new one.
	// NB we should not use the --ephemeral flag because we loose the
	//    possibility to troubleshoot. instead, we should explicitly
	//    delete the container ourselfs.
	log.Printf("Copying %s to the %s container", from, to)
	_, err = lxc("copy", from, to)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", from, to, err)
	}

	return nil
}

func lxcStart(name string) error {
	// start the container.
	log.Printf("Starting the %s container", name)
	_, err := lxc("start", name)
	if err != nil {
		return err
	}

	// wait for the container to be fully running.
	log.Printf("Waiting for the %s container to be fully running", name)
	_, err = lxc("exec", name, "--", "bash", "-c", "while [ \"$(systemctl is-system-running)\" != \"running\" ]; do sleep 1; done")
	if err != nil {
		return err
	}

	return nil
}

func lxcConfigure(name string, owner string, repo string, labels []string, token string) error {
	log.Printf("Configuring the %s runner", name)
	command := shellescape.QuoteCommand([]string{
		"/home/ghar/runner/config.sh",
		"--unattended",
		"--ephemeral",
		"--replace",
		"--url", fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		"--token", token,
		"--labels", strings.Join(labels, ","),
	})
	stdout, err := lxcWithInput(command, "exec", name, "--", "su", "-s", "/bin/bash", "-l", "ghar")
	if err != nil {
		return err
	}
	log.Printf("Configuration result:\n%s", stdout)
	return nil
}
