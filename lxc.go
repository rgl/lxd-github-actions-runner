package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
)

type writerBuffer struct {
	bytes.Buffer
}

func (b *writerBuffer) Close() error {
	return nil
}

type lxcExecError struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func (err *lxcExecError) Error() string {
	return fmt.Sprintf("failed to exec lxc: exitCode=%d stdout=%s stderr=%s", err.ExitCode, err.Stdout, err.Stderr)
}

func newLxdClient() (lxd.InstanceServer, error) {
	return lxd.ConnectLXDUnix("", nil)
}

func lxcExists(name string) (bool, error) {
	c, err := newLxdClient()
	if err != nil {
		return false, fmt.Errorf("failed to create the lxd client: %w", err)
	}
	instance, _, err := c.GetInstance(name)
	if err != nil {
		if lxdApi.StatusErrorCheck(err, 404) {
			return false, nil
		}
		return false, err
	}
	return instance != nil, nil
}

func lxcDelete(name string) error {
	c, err := newLxdClient()
	if err != nil {
		return fmt.Errorf("failed to create the lxd client: %w", err)
	}
	instance, _, err := c.GetInstance(name)
	if err != nil {
		if lxdApi.StatusErrorCheck(err, 404) {
			return nil
		}
		return err
	}
	if instance.StatusCode != 0 && instance.StatusCode != lxdApi.Stopped {
		stateRequest := lxdApi.InstanceStatePut{
			Action:  "stop",
			Timeout: -1,
			Force:   true,
		}
		op, err := c.UpdateInstanceState(name, stateRequest, "")
		if err != nil {
			return err
		}
		err = op.Wait()
		if err != nil {
			return fmt.Errorf("failed to stop instance %s: %w", name, err)
		}
		if instance.Ephemeral {
			return nil
		}
	}
	op, err := c.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete instance %s: %w", name, err)
	}
	return op.Wait()
}

func lxcExec(name string, user string, command string) error {
	path, err := exec.LookPath("lxc")
	if err != nil {
		return fmt.Errorf("failed to find lxc: %w", err)
	}
	return syscall.Exec(path, []string{"lxc", "exec", name, "--", "su", "-l", "-s", command, user}, []string{})
}

func lxcExecWithInput(name string, input string, args ...string) (string, error) {
	c, err := newLxdClient()
	if err != nil {
		return "", fmt.Errorf("failed to create the lxd client: %w", err)
	}
	execRequest := lxdApi.ContainerExecPost{
		Command:   args,
		WaitForWS: true,
	}
	execStdout := &writerBuffer{}
	execStderr := &writerBuffer{}
	execArgs := lxd.ContainerExecArgs{
		Stdin:    io.NopCloser(strings.NewReader(input)),
		Stdout:   execStdout,
		Stderr:   execStderr,
		DataDone: make(chan bool),
	}
	op, err := c.ExecContainer(name, execRequest, &execArgs)
	if err != nil {
		return "", fmt.Errorf("failed to start the exec for the container to be fully running: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return "", fmt.Errorf("failed to wait for the container to be fully running: %w", err)
	}
	opAPI := op.Get()
	<-execArgs.DataDone
	exitCode := int(opAPI.Metadata["return"].(float64))
	if exitCode != 0 {
		return "", &lxcExecError{
			ExitCode: exitCode,
			Stdout:   execStdout.String(),
			Stderr:   execStderr.String(),
		}
	}
	return strings.TrimSpace(execStdout.String()), nil
}

func lxcCopy(from, to string) error {
	// delete the "to" container if it exists.
	exists, err := lxcExists(to)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s because exists failed: %w", from, to, err)
	}
	if exists {
		log.Printf("Deleting the existing %s container", to)
		err := lxcDelete(to)
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s because delete failed: %w", from, to, err)
		}
	}

	// clone the existing container into a new one.
	// NB we should not use the --ephemeral flag because we loose the
	//    possibility to troubleshoot. instead, we should explicitly
	//    delete the container ourselfs.
	log.Printf("Copying %s to the %s container", from, to)
	c, err := newLxdClient()
	if err != nil {
		return fmt.Errorf("failed to create the lxd client: %w", err)
	}
	instance, _, err := c.GetInstance(from)
	if err != nil {
		return fmt.Errorf("failed to get instance %s: %w", from, err)
	}
	op, err := c.CopyInstance(c, *instance, &lxd.InstanceCopyArgs{Name: to})
	if err != nil {
		return fmt.Errorf("failed to copy instance %s to %s: %w", from, to, err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for copy instance %s to %s: %w", from, to, err)
	}

	return nil
}

func lxcStart(name string) error {
	c, err := newLxdClient()
	if err != nil {
		return fmt.Errorf("failed to create the lxd client: %w", err)
	}

	// start the container.
	log.Printf("Starting the %s container", name)
	op, err := c.UpdateContainerState(name, lxdApi.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}, "")
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for start container: %w", err)
	}

	// wait for the container to be fully running.
	log.Printf("Waiting for the %s container to be fully running", name)
	execRequest := lxdApi.ContainerExecPost{
		Command: []string{
			"bash",
			"-c",
			"while [ \"$(systemctl is-system-running)\" != \"running\" ]; do sleep 1; done",
		},
	}
	execArgs := lxd.ContainerExecArgs{
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	op, err = c.ExecContainer(name, execRequest, &execArgs)
	if err != nil {
		return fmt.Errorf("failed to start the exec for the container to be fully running: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for the container to be fully running: %w", err)
	}

	return nil
}
