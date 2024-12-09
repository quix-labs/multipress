package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/sync/errgroup"
	"io"
	"os/exec"
	"path/filepath"
)

func GetDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.WithAPIVersionNegotiation())
}
func ExecDockerCmd(containerName string, execOptions container.ExecOptions, inData []byte) (string, error) {
	cli, err := GetDockerClient()
	if err != nil {
		return "", fmt.Errorf("error creating Docker client: %w", err)
	}

	containerJSON, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return "", fmt.Errorf("error inspecting container: %w", err)
	}
	if !containerJSON.State.Running {
		return "", fmt.Errorf("container %s is not running", containerName)
	}

	execOptions.AttachStdout = true
	execOptions.AttachStderr = true
	execOptions.AttachStdin = true
	execOptions.Tty = false

	execIDResp, err := cli.ContainerExecCreate(context.Background(), containerName, execOptions)
	if err != nil {
		return "", fmt.Errorf("error creating exec instance: %w", err)
	}

	attachResp, err := cli.ContainerExecAttach(context.Background(), execIDResp.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    true,
	})
	if err != nil {
		return "", fmt.Errorf("error attaching to exec instance: %w", err)
	}
	defer attachResp.Close()

	err = cli.ContainerExecStart(context.Background(), execIDResp.ID, types.ExecStartCheck{Detach: false, Tty: true})
	if err != nil {
		return "", fmt.Errorf("error starting exec instance: %w", err)
	}

	var g errgroup.Group

	var inReader io.Reader
	inReader = bytes.NewReader(inData)

	if inReader != nil {
		g.Go(func() error {
			_, err := io.Copy(attachResp.Conn, inReader)
			closeErr := attachResp.CloseWrite()
			return errors.Join(err, closeErr)
		})
	}

	output := new(bytes.Buffer)
	g.Go(func() error {
		_, err = io.Copy(output, attachResp.Reader)
		return err
	})

	if err := g.Wait(); err != nil {
		return output.String(), err
	}

	execInspectResp, err := cli.ContainerExecInspect(context.Background(), execIDResp.ID)
	if err != nil {
		return output.String(), fmt.Errorf("error inspecting exec instance: %w", err)
	}

	if execInspectResp.ExitCode != 0 {
		return output.String(), fmt.Errorf("command execution failed with exit code: %d", execInspectResp.ExitCode)
	}

	return output.String(), nil
}

func GetDockerContainerIP(containerName string) (string, error) {
	cli, err := GetDockerClient()
	if err != nil {
		return "", err
	}

	containerInspect, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return "", fmt.Errorf("unable to inspect container %s: %v", containerName, err)
	}

	for _, network := range containerInspect.NetworkSettings.Networks {
		if network.IPAddress != "" {
			return network.IPAddress, nil
		}
	}

	return "", fmt.Errorf("unable to retrieve IP address for container %s", containerName)
}

func UpComposeFile(composeFilePath string) (string, error) {
	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "up", "-d", "--wait")
	cmd.Dir = filepath.Dir(composeFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to deploy %s: %w. Output: %s", composeFilePath, err, string(output))
	}
	return string(output), nil
}

func DownComposeFile(composeFilePath string) (string, error) {
	cmd := exec.Command("docker", "compose", "-f", composeFilePath, "down")
	cmd.Dir = filepath.Dir(composeFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to down %s: %w. Output: %s", composeFilePath, err, string(output))
	}
	return string(output), nil
}
