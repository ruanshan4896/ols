package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type DockerManager struct{}

func NewDockerManager() *DockerManager {
	return &DockerManager{}
}

func buildComposeArgs(dir string, subCmd ...string) []string {
	args := []string{"compose", "--project-directory", dir}
	return append(args, subCmd...)
}

func (d *DockerManager) EnsureNetwork(name string) error {
	checkCmd := exec.Command("docker", "network", "inspect", name)
	if err := checkCmd.Run(); err == nil {
		return nil // Network already exists
	}

	createCmd := exec.Command("docker", "network", "create", name)
	if out, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create docker network %s: %s (%w)", name, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeUp(dir string) error {
	args := buildComposeArgs(dir, "up", "-d", "--remove-orphans")
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose up in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeDown(dir string, removeVolumes bool) error {
	subCmd := []string{"down"}
	if removeVolumes {
		subCmd = append(subCmd, "-v")
	}
	args := buildComposeArgs(dir, subCmd...)
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose down in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeRestart(dir string) error {
	args := buildComposeArgs(dir, "restart")
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose restart in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ExecInContainer(container string, command ...string) (string, error) {
	args := append([]string{"exec", container}, command...)
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (d *DockerManager) IsContainerRunning(container string) bool {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", container)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) == "true"
}
